// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package zapavm

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
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
	MockZcash = false
	TestNet = true
)

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
	log.Info("Initializing zapa VM", "Version", version, "nodeid", ctx.NodeID, "config", configData)
	vm.dbManager = dbManager
	vm.ctx = ctx
	vm.toEngine = toEngine
	vm.verifiedBlocks = make(map[ids.ID]*Block)
	vm.as = as
	var conf VMConfig
	jerr := nativejson.Unmarshal(configData, &conf)
	vm.zc = getZCashClient(ctx, conf, jerr == nil)
	// Create new state
	vm.state = NewState(vm.dbManager.Current().Database, vm)

	// Initialize genesis
	if err := vm.initAndSync(genesisData); err != nil {
		return err
	}

	// Get last accepted
	lastAccepted, err := vm.state.GetLastAccepted()
	if err != nil {
		return err
	}

	ctx.Log.Info("initializing last accepted block as %s", lastAccepted)

	// Build off the most recently accepted block
	return vm.SetPreference(lastAccepted)
}

func getZCashClient(ctx *snow.Context, conf VMConfig, useConf bool) zclient.ZcashClient {
	if MockZcash {
		log.Info("Using mock zcash client")
		return zclient.NewDefaultMock()
	}
    if ! useConf {
		// try reading a custom file
		h, _ := os.LookupEnv("HOME")
		plan, _ := ioutil.ReadFile(h +  "/.avalanchego/configs/vms/zapavm/node.json")
		var data map[string]interface{}
		log.Info("Attempting to marshal", "contents", plan)
		err := nativejson.Unmarshal(plan, &data)
		if err != nil {
			log.Info("error reading local node config...getting node config from file based onour node number", "e", err)
			i := 0
			for i < 6 {
				fname := h + "/node-ids/" + strconv.Itoa(i)
				nid, _ := ioutil.ReadFile(fname)
				snid := strings.ReplaceAll(string(nid), "NodeID-", "")
				log.Info("comparing", "file name", fname, "fvalue", snid, "nid", ctx.NodeID.String())
				if snid == ctx.NodeID.String() {
					log.Info("Initializing zcash client as node num", "num", i)
					return &zclient.ZcashHTTPClient {
						Host: "127.0.0.1",
						Port: 8232 + i,
						User: "test",
						Password: "pw",
					}
				}
				i += 1
			}
		} else {
			log.Info("successfuly sourced node config", "config", data)
			return &zclient.ZcashHTTPClient {
				Host: data["zc_host"].(string),
				Port: data["zc_port"].(int),
				User: data["zc_user"].(string),
				Password: data["zc_password"].(string),
			}
		}
    } 
	log.Info("Using default client")
	return &zclient.ZcashHTTPClient{
		Host:conf.ZcashHost,
		Port: conf.ZcashPort,
		User: conf.ZcashUser,
		Password: conf.ZcashPassword,
	}
}

// SetState sets this VM state according to given snow.State
func (vm *VM) SetState(state snow.State) error {
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

// VerifyHeightIndex should return:
// - nil if the height index is available.
// - ErrHeightIndexedVMNotImplemented if the height index is not supported.
// - ErrIndexIncomplete if the height index is not currently available.
// - Any other non-standard error that may have occurred when verifying the
//   index.
func (vm *VM) VerifyHeightIndex() error {
	log.Info("verify height index")
	return nil
}

// GetBlockIDAtHeight returns the ID of the block that was accepted with
// [height].
func (vm *VM) GetBlockIDAtHeight(height uint64) (ids.ID, error) {
	log.Info("get block id at height", "height", height)
	return vm.state.GetBlockIDAtHeight(height)
}

func (vm *VM) GetBlockAtHeight (height uint64) (snowman.Block, error) {
	blockId, e := vm.GetBlockIDAtHeight(height)
	if e != nil {
		return &Block{}, fmt.Errorf("error getting block at height %d : %e", height, e)
	}
	return vm.GetBlock(blockId)
}

// onBootstrapStarted marks this VM as bootstrapping
func (vm *VM) onBootstrapStarted() error {
	log.Info("on bootstrap started")
	vm.bootstrapped.SetValue(false)
	return nil
}

// onNormalOperationsStarted marks this VM as bootstrapped
func (vm *VM) onNormalOperationsStarted() error {
	// No need to set it again
	log.Info("on normal operations started")
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
func (vm *VM) initAndSync(genesisData []byte) error {
	log.Info("Entering initAndSync function")
	stateInitialized, err := vm.state.IsInitialized()
	if err != nil {
		return err
	}
	zcBlkCount, err := vm.zc.GetBlockCount()
	if err != nil {
		log.Error("Error getting block count from zcash", err)
		return err
	}	
	
	preferredBlock, err := vm.getBlock(vm.preferred)
	if err != nil {
		return fmt.Errorf("couldn't get preferred block: %w", err)
	}
	preferredHeight := int(preferredBlock.Height())

	log.Info("got heights", "zapavm height", preferredHeight, "zcash height", zcBlkCount)


	if stateInitialized {

		if zcBlkCount > preferredHeight {
			return fmt.Errorf("Cannot initialize vm when zcash has existing blocks this VM doesn't know about")
		} 

		for preferredHeight > zcBlkCount {
			zcBlkCount += 1
			log.Info("Syncing block with zcash", "block number", zcBlkCount)
			snoblk, e := vm.GetBlockAtHeight(uint64(zcBlkCount))
			if e != nil {
				return e
			}
			blk, ok := snoblk.(*Block)
			if !ok {
				return fmt.Errorf("Failed to cast snowman block to Block")
			}
			e = vm.zc.SubmitBlock(blk.ZBlock())
			if e != nil {
				return fmt.Errorf("error while submitting block when syncing zcash %e", e)
			}
		}
	} else {
		// Initialize
		log.Info("Initializing zapavm by ingesting existing zcash blocks")
		var height uint64 = 0
		parentid := ids.Empty
		for blk := range zclient.BlockGenerator(vm.zc) {
			if blk.Error != nil {
				return fmt.Errorf("Error when retrieving block from zcash %e", blk.Error)
			}
			zapablk, err := vm.NewBlock(parentid, height, blk.Block, blk.Timestamp)
			if err != nil {
				return err
			}
			zapablk.Accept()
			parentid = zapablk.ID()
			height++
		}
	}

	// set state as initialized
	if err := vm.state.SetInitialized(); err != nil {
		log.Error("error while setting db to initialized: %w", err)
		return err
	}

	log.Info("finished initialization, committing initialized state")
	return vm.state.Commit()
}

// CreateHandlers returns a map where:
// Keys: The path extension for this VM's API (empty in this case)
// Values: The handler for the API
func (vm *VM) CreateHandlers() (map[string]*common.HTTPHandler, error) {
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
	newBlock, err := vm.NewBlock(vm.preferred, preferredHeight+1, suggestResult.Block, suggestResult.Timestamp)
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

// ParseBlock parses [bytes] to a Block
// This function is used by the vm's state to unmarshal blocks saved in state
// and by the consensus layer when it receives the byte representation of a block
// from another node
func (vm *VM) ParseBlock(bytes []byte) (snowman.Block, error) {
	// A new empty block
	block := &Block{}

	// Unmarshal the byte repr. of the block into our empty block
	_, err := Codec.Unmarshal(bytes, block)
	if err != nil {
		return nil, err
	}

	// Initialize the block
	block.Initialize(bytes, choices.Processing, vm)

	if blk, err := vm.getBlock(block.ID()); err == nil {
		// If we have seen this block before, return it with the most up-to-date
		// info
		return blk, nil
	}

	// Return the block
	return block, nil
}

// NewBlock returns a new Block where:
// - the block's parent is [parentID]
// - the block's data is [data]
// - the block's timestamp is [timestamp]
func (vm *VM) NewBlock(parentID ids.ID, height uint64, zblock nativejson.RawMessage, timestamp int64) (*Block, error) {
	block := &Block{
		PrntID: parentID,
		Hght:   height,
		ZBlk:   zblock,
		timestamp: timestamp,
	}

	// Get the byte representation of the block
	blockBytes, err := Codec.Marshal(CodecVersion, block)
	if err != nil {
		return nil, err
	}

	// Initialize the block by providing it with its byte representation
	// and a reference to this VM
	block.Initialize(blockBytes, choices.Processing, vm)
	return block, nil
}

// Shutdown this vm
func (vm *VM) Shutdown() error {
	if vm.state == nil {
		return nil
	}

	return vm.state.Close() // close versionDB
}

// SetPreference sets the block with ID [ID] as the preferred block
func (vm *VM) SetPreference(id ids.ID) error {
	vm.preferred = id
	return nil
}

// Bootstrapped marks this VM as bootstrapped
func (vm *VM) Bootstrapped() error { 
	log.Info("node finished bootstrapping", "node id", vm.ctx.NodeID)
	return nil 
}

// Bootstrapping marks this VM as bootstrapping
func (vm *VM) Bootstrapping() error { 
	log.Info("node is bootstrapping", "node id", vm.ctx.NodeID)
	return nil 
}

// Returns this VM's version
func (vm *VM) Version() (string, error) {
	return Version.String(), nil
}

func (vm *VM) Connected(id ids.ShortID, v version.Application) error {
	log.Debug("Connected to node id", "node id", id, "app version", v.String())
	return nil // noop
}

func (vm *VM) Disconnected(id ids.ShortID) error {
	return nil // noop
}

// Receive transaction
func (vm *VM) AppGossip(nodeID ids.ShortID, msg []byte) error {
	// receive gossip, add to mempool
	log.Info("receiving app gossip", "fromnodeid", nodeID, "receivingnodeid", vm.ctx.NodeID)
	if msg != nil {
		log.Info("calling zcash receive tx", "fromnodeid", nodeID, "receivingnodeid", vm.ctx.NodeID)
		vm.zc.CallZcash("receivetx", msg)
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