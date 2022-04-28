// (c) 2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package zapavm

import (
	"github.com/ava-labs/avalanchego/chains/atomic"
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/database/prefixdb"
	"github.com/ava-labs/avalanchego/database/versiondb"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	pstate "github.com/ava-labs/avalanchego/vms/proposervm/state"
	log "github.com/inconshreveable/log15"
)

var (
	// These are prefixes for db keys.
	// It's important to set different prefixes for each separate database objects.
	singletonStatePrefix = []byte("singleton")
	blockStatePrefix     = []byte("block")
	heightIndexPrefix    = []byte("height")

	_ State = &state{}
)

// State is a wrapper around avax.SingleTonState and BlockState
// State also exposes a few methods needed for managing database commits and close.
type State interface {
	// SingletonState is defined in avalanchego,
	// it is used to understand if db is initialized already.
	avax.SingletonState
	BlockState
	pstate.HeightIndex

	Commit() error
	Close() error
	ClearState() error
}

type state struct {
	avax.SingletonState
	BlockState
	pstate.HeightIndex

	baseDB *versiondb.Database
}

func (s *state) PutBlock(blk *Block) error {
	log.Info("state.Putblock");
	pberr := s.BlockState.PutBlock(blk)
	if pberr != nil {
		return pberr
	}
	if blk.Status() == choices.Accepted {
		return s.HeightIndex.SetBlockIDAtHeight(blk.Hght, blk.id)
	}
	return nil
}

func DeleteDb(db database.Database) error {
	dataBatch := db.NewBatch()
	var err error
	it := db.NewIterator()
	defer it.Release()
	deletedItems := 0

	for it.Next() {
		if err = dataBatch.Delete(it.Key()); err != nil {
			log.Error("Error deleting key", "key", it.Key(), "error", err)
			return err
		}
		log.Info("deleted key", "key", it.Key())
		deletedItems += 1
	}

	if err = it.Error(); err != nil {
		return err
	}

	if err := atomic.WriteAll(dataBatch); err != nil {
		log.Error("Error applying batch", "error", err)
		return err
	}
	log.Info("Successfully deleted", "num keys", deletedItems)
	return nil
}


func NewState(db database.Database, vm *VM) State {
	// create a new baseDB
	baseDB := versiondb.New(db)

	chainPrefix := vm.ctx.ChainID.String()

	// create a prefixed "blockDB" from baseDB
	blockDBPref := chainPrefix + "-" + string(blockStatePrefix)
	singletonDBPref := chainPrefix + "-" + string(singletonStatePrefix)
	heightDBPref := chainPrefix + "-" + string(heightIndexPrefix)


	blockDB := prefixdb.New([]byte(blockDBPref), baseDB)
	singletonDB := prefixdb.New([]byte(singletonDBPref), baseDB)

	heightDB := prefixdb.New([]byte(heightDBPref), baseDB)

	// return state with created sub state components
	return &state{
		BlockState:     NewBlockState(blockDB, vm),
		SingletonState: avax.NewSingletonState(singletonDB),
		HeightIndex:    pstate.NewHeightIndex(heightDB, baseDB),
		baseDB:         baseDB,
	}
}

// Commit commits pending operations to baseDB
func (s *state) Commit() error {
	return s.baseDB.Commit()
}

// Close closes the underlying base database
func (s *state) Close() error {
	return s.baseDB.Close()
}

func (s *state) ClearState() error {
	log.Info("clearing state...")
	return DeleteDb(s.baseDB)
}
