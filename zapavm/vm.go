// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package zapavm

import (
	"fmt"
	"os"
	"time"

	"github.com/gorilla/rpc/v2"
	log "github.com/inconshreveable/log15"
	"github.com/zapalabs/zapavm/zapavm/zclient"

	nativejson "encoding/json"

	"github.com/ava-labs/avalanchego/database/manager"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/snow/engine/snowman/block"
	"github.com/ava-labs/avalanchego/utils"
	"github.com/ava-labs/avalanchego/utils/json"
	"github.com/ava-labs/avalanchego/version"
	"github.com/ava-labs/avalanchego/vms/proposervm/indexer"
)

const (
	dataLen = 32
	Name    = "zapavm"
)

var (
	Version            = version.NewDefaultVersion(1, 2, 0)
	_ block.ChainVM = &VM{}

	// Whether or not we're on fuji. Controls whether or not certain
	// debug features were enabled (e.g. faucet, mining empty blocks)
	TestNet = true

	// If a chain is here, it's disabled. If the disabled flag is present
	// in the chain config, the chain will also be disabled. A chain cannot
	// be removed from this list via config.
	DisabledChains = map[string]bool{
		"HydwMTPrYBWHrGVmWfG8k4Po2eTPEqe7y7Z4jZaUr2Me6rin7": true,
		"2LedqoeDb3zZQSqPBczemzrofepr6SzSHXxHXANrfzeFKGGNVd": true,
	}
)

var originalStderr *os.File

func init() {
	// Preserve [os.Stderr] prior to the call in plugin/main.go to plugin.Serve(...).
	// Preserving the log level allows us to update the root handler while writing to the original
	// [os.Stderr] that is being piped through to the logger via the rpcchainvm.
	originalStderr = os.Stderr
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlDebug, log.StreamHandler(originalStderr, log.TerminalFormat())))
}

// VM implements the snowman.VM interface
// Each block in this chain contains a Unix timestamp
// and a piece of data (a string)
type VM struct {
	// The context of this vm
	ctx       *snow.Context
	dbManager manager.Manager

	// State of this VM
	state State

	// ID of the preferred block
	preferred ids.ID

	// channel to send messages to the consensus engine
	toEngine chan<- common.Message

	// Block ID --> Block
	// Each element is a block that passed verification but
	// hasn't yet been accepted/rejected
	verifiedBlocks map[ids.ID]*Block

	hIndexer indexer.HeightIndexer

	as common.AppSender

	zc zclient.ZcashClient

	enabled bool

	// Indicates that this VM has finised bootstrapping for the chain
	bootstrapped utils.AtomicBool
}

// Initialize this vm
// [ctx] is this vm's context
// [dbManager] is the manager of this vm's database
// [toEngine] is used to notify the consensus engine that new blocks are
//   ready to be added to consensus
// The data in the genesis block is [genesisData]
func (vm *VM) Initialize(
	ctx *snow.Context,
	dbManager manager.Manager,
	genesisData []byte,
	upgradeData []byte,
	configData []byte,
	toEngine chan<- common.Message,
	_ []*common.Fx,
	as common.AppSender,
) error {
	version, err := vm.Version()
	if err != nil {
		log.Error("error initializing Zapa VM: %v", err)
		return err
	}	
	vm.dbManager = dbManager
	vm.ctx = ctx
	vm.toEngine = toEngine
	vm.verifiedBlocks = make(map[ids.ID]*Block)
	vm.as = as
	conf := NewChainConfig(configData)

	logLevel, err := log.LvlFromString(conf.LogLevel)
	if err != nil {
		return fmt.Errorf("failed to initialize logger due to: %w ", err)
	}
	vm.setLogLevel(logLevel)

	log.Info("Initializing zapa VM", "Version", version, "nodeid", ctx.NodeID, "config", conf)

	vm.enabled = vm.isEnabled(conf)

	if !vm.enabled {
		return fmt.Errorf("Chain %s is not enabled", vm.ctx.ChainID)
	}

	vm.zc, err = conf.ZcashClient(vm.ctx.NodeID.String())

	if err != nil {
		return fmt.Errorf("Error initializing zcash client: %e", err)
	}

	// Create new state
	vm.state = NewState(vm.dbManager.Current().Database, vm)

	if conf.ClearDatabase {
		log.Info("Clearing database before initializing...")
		err := vm.state.ClearState()
		if err != nil {
			return err
		}
	} else {
		log.Debug("Not clearing database before initializing, picking up where we left off...")
	}
	res := vm.initAndSync()
	if res != nil {
		log.Error("Error during initialization", "error", res)
	} else {
		log.Info("Successfully completed initialization of zapavm")
	}
	return res
}

// SetState sets this VM state according to given snow.State
func (vm *VM) SetState(state snow.State) error {
	log.Info("Setting state", "state", state)
	switch state {
	// Engine reports it's bootstrapping
	case snow.Bootstrapping:
		return vm.onBootstrapStarted()
	case snow.NormalOp:
		// Engine reports it can start normal operations
		return vm.onNormalOperationsStarted()
	default:
		return snow.ErrUnknownState
	}
}

// setLogLevel initializes logger and sets the log level with the original [os.StdErr] interface
// along with the context logger.
func (vm *VM) setLogLevel(logLevel log.Lvl) {
	prefix, err := vm.ctx.BCLookup.PrimaryAlias(vm.ctx.ChainID)
	if err != nil {
		prefix = vm.ctx.ChainID.String()
	}
	prefix = fmt.Sprintf("<%s Chain>", prefix)
	format := SubnetEVMFormat(prefix)
	log.Root().SetHandler(log.LvlFilterHandler(logLevel, log.MultiHandler(
		log.StreamHandler(originalStderr, format),
		log.StreamHandler(vm.ctx.Log, format),
	)))
}

func SubnetEVMFormat(prefix string) log.Format {
	return log.FormatFunc(func(r *log.Record) []byte {
		location := fmt.Sprintf("%+v", r.Call)
		newMsg := fmt.Sprintf("%s %s: %s", prefix, location, r.Msg)
		// need to deep copy since we're using a multihandler
		// as a result it will alter R.msg twice.
		newRecord := log.Record{
			Time:     r.Time,
			Lvl:      r.Lvl,
			Msg:      newMsg,
			Ctx:      r.Ctx,
			Call:     r.Call,
			KeyNames: r.KeyNames,
		}
		b := log.TerminalFormat().Format(&newRecord)
		return b
	})
}

// VerifyHeightIndex should return:
// - nil if the height index is available.
// - ErrHeightIndexedVMNotImplemented if the height index is not supported.
// - ErrIndexIncomplete if the height index is not currently available.
// - Any other non-standard error that may have occurred when verifying the
//   index.
func (vm *VM) VerifyHeightIndex() error {
	log.Debug("VerifyHeightIndex invoked")
	return nil
}

// GetBlockIDAtHeight returns the ID of the block that was accepted with
// [height].
func (vm *VM) GetBlockIDAtHeight(height uint64) (ids.ID, error) {
	log.Info("get block id at height", "height", height)
	return vm.state.GetBlockIDAtHeight(height)
}

func (vm *VM) GetBlockAtHeight (height uint64) (*Block, error) {
	blockId, e := vm.GetBlockIDAtHeight(height)
	if e != nil {
		return &Block{}, fmt.Errorf("error getting block at height %d : %e", height, e)
	}
	return vm.state.GetBlock(blockId)
}

// CreateHandlers returns a map where:
// Keys: The path extension for this VM's API (empty in this case)
// Values: The handler for the API
func (vm *VM) CreateHandlers() (map[string]*common.HTTPHandler, error) {
	log.Info("creating handlers")
	server := rpc.NewServer()
	server.RegisterCodec(json.NewCodec(), "application/json")
	server.RegisterCodec(json.NewCodec(), "application/json;charset=UTF-8")
	if err := server.RegisterService(&Service{vm: vm}, Name); err != nil {
		return nil, err
	}

	return map[string]*common.HTTPHandler{
		"": {
			Handler: server,
		},
	}, nil
}

// CreateStaticHandlers returns a map where:
// Keys: The path extension for this VM's static API
// Values: The handler for that static API
func (vm *VM) CreateStaticHandlers() (map[string]*common.HTTPHandler, error) {
	log.Info("creating static handlers")
	server := rpc.NewServer()
	server.RegisterCodec(json.NewCodec(), "application/json")
	server.RegisterCodec(json.NewCodec(), "application/json;charset=UTF-8")
	if err := server.RegisterService(&StaticService{}, Name); err != nil {
		return nil, err
	}

	return map[string]*common.HTTPHandler{
		"": {
			LockOptions: common.NoLock,
			Handler:     server,
		},
	}, nil

}

// Health implements the common.VM interface
func (vm *VM) HealthCheck() (interface{}, error) { return nil, nil }

// BuildBlock returns a block that this vm wants to add to consensus
func (vm *VM) BuildBlock() (snowman.Block, error) {
	log.Info("vm.BuildBlock: begin. Building and proposing block for consensus")
	suggestResult := vm.zc.SuggestBlock()
	if suggestResult.Error != nil {
		return nil, fmt.Errorf("Error suggesting block %e", suggestResult.Error)
	}

	// Gets Preferred Block
	preferredBlock, err := vm.getBlock(vm.preferred)
	if err != nil {
		return nil, fmt.Errorf("couldn't get preferred block: %w", err)
	}
	preferredHeight := preferredBlock.Height()

	// Build the block with preferred height
	newBlock, err := vm.NewBlock(vm.preferred, preferredHeight+1, suggestResult.Block, suggestResult.Timestamp, suggestResult.Hash, suggestResult.ParentHash)
	if err != nil {
		return nil, fmt.Errorf("couldn't build block: %w", err)
	}

	// Verifies block
	if err := newBlock.Verify(); err != nil {
		return nil, err
	}
	return newBlock, nil
}

// NotifyBlockReady tells the consensus engine that a new block
// is ready to be created
func (vm *VM) NotifyBlockReady() {
	select {
	case vm.toEngine <- common.PendingTxs:
	default:
		vm.ctx.Log.Debug("dropping message to consensus engine")
	}
}

// GetBlock implements the snowman.ChainVM interface
func (vm *VM) GetBlock(blkID ids.ID) (snowman.Block, error) { return vm.getBlock(blkID) }

func (vm *VM) getBlock(blkID ids.ID) (*Block, error) {
	// If block is in memory, return it.
	if blk, exists := vm.verifiedBlocks[blkID]; exists {
		return blk, nil
	}

	return vm.state.GetBlock(blkID)
}

// LastAccepted returns the block most recently accepted
func (vm *VM) LastAccepted() (ids.ID, error) { return vm.state.GetLastAccepted() }

// LastAccepted returns the block most recently accepted
func (vm *VM) LastAcceptedBlock() (*Block, error) { 
	id, err := vm.LastAccepted()
	if err != nil {
		return nil, fmt.Errorf("Error getting last accepted block id %e", err)
	}
	return vm.getBlock(id)
}

// ParseBlock parses [bytes] to a Block
// This function is used by the vm's state to unmarshal blocks saved in state
// and by the consensus layer when it receives the byte representation of a block
// from another node
func (vm *VM) ParseBlock(bytes []byte) (snowman.Block, error) {
	log.Debug("ParseBlock: begin")
	// A new empty block
	block := &Block{}

	// Unmarshal the byte repr. of the block into our empty block
	_, err := Codec.Unmarshal(bytes, block)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling block: %e", err)
	}

	// Initialize the block
	block.Initialize(bytes, choices.Processing, vm)

	if blk, err := vm.getBlock(block.ID()); err == nil {
		// If we have seen this block before, return it with the most up-to-date
		// info
		log.Debug("ParseBlock: return. We have seen this block", blk.LogInfo()...)
		return blk, nil
	}

	log.Debug("ParseBlock: return. Returning block", block.LogInfo()...)
	// Return the block
	return block, nil
}

// NewBlock returns a new Block where:
// - the block's parent is [parentID]
// - the block's data is [data]
// - the block's timestamp is [timestamp]
func (vm *VM) NewBlock(parentID ids.ID, height uint64, zblock nativejson.RawMessage, timestamp int64, hash string, parentHash string) (*Block, error) {
	log.Debug("NewBlock: begin")
	block := &Block{
		PrntID: parentID,
		Hght:   height,
		ZBlk:   zblock,
		CreationTime: timestamp,
		ZHash: hash,
		ZParent: parentHash,
	}

	if height > 0 {
		block.ProducingNode = vm.ctx.NodeID.String()
	}

	// Get the byte representation of the block
	blockBytes, err := Codec.Marshal(CodecVersion, block)
	if err != nil {
		return nil, fmt.Errorf("Error marshalling block bytes %e", err)
	}

	// Initialize the block by providing it with its byte representation
	// and a reference to this VM
	block.Initialize(blockBytes, choices.Processing, vm)
	log.Debug("NewBlock: return. Returning block", block.LogInfo()...)
	return block, nil
}

// Shutdown this vm
func (vm *VM) Shutdown() error {
	log.Debug("Shutdown: begin")
	if vm.state == nil {
		return nil
	}

	log.Debug("Shutdown: calling vm.state.close()")
	return vm.state.Close() // close versionDB
}

// SetPreference sets the block with ID [ID] as the preferred block
func (vm *VM) SetPreference(id ids.ID) error {
	log.Debug("SetPreference: begin", "id", id)
	vm.preferred = id
	return nil
}

// Returns this VM's version
func (vm *VM) Version() (string, error) {
	return Version.String(), nil
}

func (vm *VM) Connected(id ids.ShortID, v version.Application) error {
	log.Debug("Connected to node id", "node id", id, "app version", v.String())
	return nil
}

func (vm *VM) Disconnected(id ids.ShortID) error {
	log.Debug("Disconnected from node id", "node id", id)
	return nil 
}

// Receive transaction
func (vm *VM) AppGossip(nodeID ids.ShortID, msg []byte) error {
	log.Debug("Receiving app gossip", "fromNodeID", nodeID, "receivingNodeID", vm.ctx.NodeID)
	if msg != nil {
		log.Debug("Calling zcash.receivetx")
		vm.zc.CallZcash("receivetx", msg)
		log.Debug("Calling vm.NotifyBlockReady()")
		vm.NotifyBlockReady()
	}

	return nil
}

// This VM doesn't (currently) have any app-specific messages
func (vm *VM) AppRequest(nodeID ids.ShortID, requestID uint32, time time.Time, request []byte) error {
	return nil
}

// This VM doesn't (currently) have any app-specific messages
func (vm *VM) AppResponse(nodeID ids.ShortID, requestID uint32, response []byte) error {
	return nil
}

// This VM doesn't (currently) have any app-specific messages
func (vm *VM) AppRequestFailed(nodeID ids.ShortID, requestID uint32) error {
	return nil
}

func (vm *VM) Commit() error {
	vm.ctx.Lock.Lock()
	defer vm.ctx.Lock.Unlock()

	return vm.state.Commit()
}

// onBootstrapStarted marks this VM as bootstrapping
func (vm *VM) onBootstrapStarted() error {
	log.Info("Bootstrapping started...")
	vm.bootstrapped.SetValue(false)
	return nil
}

// onNormalOperationsStarted marks this VM as bootstrapped
func (vm *VM) onNormalOperationsStarted() error {
	log.Info("Normal Operations Started")
	if vm.bootstrapped.GetValue() {
		return nil
	}
	vm.bootstrapped.SetValue(true)
	return nil
}

// Sync this node with the zcash daemon
// If we are initializing, ingest the genesis block from the zcash daemon
// Otherwise, if our chain is ahead of the zcash daemon's chain, catch up the daemon
// All other states result in an exception
func (vm *VM) initAndSync() error {
	log.Debug("initAndSync: begin")
	stateInitialized, err := vm.state.IsInitialized()
	if err != nil {
		return err
	}
	zcBlkCount, err := vm.zc.GetBlockCount()
	if err != nil {
		log.Error("Error getting block count from zcash", err)
		return err
	}	
	
	if stateInitialized {
		err := vm.initializePreference()
		if err != nil {
			return fmt.Errorf("error initializing preference %e", err)
		}

		preferredBlock, err := vm.getBlock(vm.preferred)
		if err != nil {
			log.Error("Couldn't get preferred block")
			return fmt.Errorf("couldn't get preferred block: %w", err)
		}
		preferredHeight := int(preferredBlock.Height())

		log.Info("Chain has alreay been initialized", "current zcash height", zcBlkCount, "current zapavm height", preferredHeight)
		
		if zcBlkCount > preferredHeight {
			return fmt.Errorf("Cannot initialize vm when zcash has existing blocks this VM doesn't know about")
		} 

		for preferredHeight > zcBlkCount {
			zcBlkCount += 1
			blk, e := vm.GetBlockAtHeight(uint64(zcBlkCount))
			log.Info("Syncing block with zcash", "block number", zcBlkCount, "zbytes", blk.ZBlock())
			if e != nil {
				return e
			}
			e = blk.Verify()
			if e != nil {
				log.Info("Error. Deleting this and all subsequent blocks", "error", e, "height", blk.Height())
				for blk != nil {
					blk.Reject()
					blk, _ = vm.GetBlockAtHeight(uint64(zcBlkCount))
				}
				break
			}
			e = vm.zc.SubmitBlock(blk.ZBlock())
			if e != nil {
				return fmt.Errorf("error while submitting block when syncing zcash %e", e)
			}
			blk.Refresh()
			vm.state.PutBlock(blk)
		}
	} else {
		log.Info("Initializing zapavm by ingesting genesis from zcash")

		var height uint64 = 0
		parentid := ids.Empty
		for blk := range zclient.BlockGenerator(vm.zc) {

			if height == 1 {
				return fmt.Errorf("Initializing zapavm with zcash which has more than genesis block. This is unacceptable!")
			}
			if blk.Error != nil {
				return fmt.Errorf("Error when retrieving block from zcash %e", blk.Error)
			}
			zapablk, err := vm.NewBlock(parentid, height, blk.Block, blk.Timestamp, blk.Hash, blk.ParentHash)
			if err != nil {
				return err
			}
			log.Info("Build genesis block", zapablk.LogInfo()...)
			zapablk.Accept()
			log.Info("Accepted genesis block", zapablk.LogInfo()...)
			parentid = zapablk.ID()
			height++
		}

		err := vm.initializePreference()
		if err != nil {
			return fmt.Errorf("error initializing preference %e", err)
		}
	}

	// set state as initialized
	if err := vm.state.SetInitialized(); err != nil {
		log.Error("error while setting db to initialized: %w", err)
		return err
	}

	log.Info("initAndSync: finished initialization, Committing initialized state")
	return vm.state.Commit()
}

// min is inclusive, max is exclusive
func (vm *VM) BlockGenerator(min *int, max *int) chan Block {
	c := make(chan Block)
	start := 0
	if min != nil {
		start = *min
	}
	
	go func() {
		defer close(c)
		var lastBlock *Block
		var err error
		lastBlock, err = vm.state.GetLastAcceptedBlock()
		if err != nil {
			log.Error("Error getting last accepted block", "error", err)
			return
		}
		if max != nil && *max < int(lastBlock.Height()) {
			lastBlock, err = vm.GetBlockAtHeight(uint64(*max))
			if err != nil {
				log.Error("Error getting block at height", "height", max, "error", err)
				return
			}
		}
		if start > int(lastBlock.Height()) {
			log.Error("Requesting all blocks above height that chain hasn't reached")
			return
		}
		var currBlock *Block
		height := start
		for currBlock == nil || !(currBlock.ID().String() == lastBlock.ID().String()) {
			currBlock, err = vm.GetBlockAtHeight(uint64(height))
			if err != nil {
				log.Error("Error retrieving block at height", "error", err)
				return
			}
			c <- *currBlock
			height++
		}
	}()

	return c
}

func (vm *VM) isEnabled(c ChainConfig) bool {
	if _, ok := DisabledChains[vm.ctx.ChainID.String()]; ok {
		return false
	}
	return c.Enabled
}

func (vm *VM) initializePreference() error {
	// Get last accepted
	lastAccepted, err := vm.state.GetLastAccepted()
	if err != nil {
		return fmt.Errorf("Error getting last accepted block %e", err)
	}
	err = vm.SetPreference(lastAccepted)
	if err != nil {
		return fmt.Errorf("Error setting preference %e", err)
	}
	return nil
}