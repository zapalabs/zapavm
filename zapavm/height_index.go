package zapavm

import (
	"github.com/ava-labs/avalanchego/cache"
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/database/prefixdb"
	"github.com/ava-labs/avalanchego/database/versiondb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/wrappers"
	pstate "github.com/ava-labs/avalanchego/vms/proposervm/state"

)

const cacheSize = 8192

var (

	heightPrefix   = []byte("height")
	metadataPrefix = []byte("metadata")

	forkKey       = []byte("fork")
	checkpointKey = []byte("checkpoint")
)

type HeightIndex interface {
	pstate.HeightIndex

	DeleteBlockAtHeight(height uint64) error
}

type ZapaHeightIndex struct {
	versiondb.Commitable

	// Caches block height -> proposerVMBlockID.
	heightsCache cache.Cacher

	heightDB   database.Database
	metadataDB database.Database
}

func NewZapaHeightIndex(db database.Database, commitable versiondb.Commitable) *ZapaHeightIndex {
	return &ZapaHeightIndex{
		Commitable: commitable,

		heightsCache: &cache.LRU{Size: cacheSize},
		heightDB:     prefixdb.New(heightPrefix, db),
		metadataDB:   prefixdb.New(metadataPrefix, db),
	}
}


func (hi *ZapaHeightIndex) ResetHeightIndex() error {
	var (
		itHeight   = hi.heightDB.NewIterator()
		itMetadata = hi.metadataDB.NewIterator()
	)
	defer func() {
		itHeight.Release()
		itMetadata.Release()
	}()

	// clear height cache
	hi.heightsCache.Flush()

	// clear heightDB
	for itHeight.Next() {
		if err := hi.heightDB.Delete(itHeight.Key()); err != nil {
			return err
		}
	}

	// clear metadataDB
	for itMetadata.Next() {
		if err := hi.metadataDB.Delete(itMetadata.Key()); err != nil {
			return err
		}
	}

	errs := wrappers.Errs{}
	errs.Add(
		itHeight.Error(),
		itMetadata.Error(),
	)
	return errs.Err
}

func (hi *ZapaHeightIndex) GetBlockIDAtHeight(height uint64) (ids.ID, error) {
	if blkIDIntf, found := hi.heightsCache.Get(height); found {
		res, _ := blkIDIntf.(ids.ID)
		return res, nil
	}

	key := database.PackUInt64(height)
	blkID, err := database.GetID(hi.heightDB, key)
	if err != nil {
		return ids.Empty, err
	}
	hi.heightsCache.Put(height, blkID)
	return blkID, err
}

func (hi *ZapaHeightIndex) SetBlockIDAtHeight(height uint64, blkID ids.ID) error {
	hi.heightsCache.Put(height, blkID)
	key := database.PackUInt64(height)
	return database.PutID(hi.heightDB, key, blkID)
}

func (hi *ZapaHeightIndex) GetForkHeight() (uint64, error) {
	return database.GetUInt64(hi.metadataDB, forkKey)
}

func (hi *ZapaHeightIndex) SetForkHeight(height uint64) error {
	return database.PutUInt64(hi.metadataDB, forkKey, height)
}

func (hi *ZapaHeightIndex) GetCheckpoint() (ids.ID, error) {
	return database.GetID(hi.metadataDB, checkpointKey)
}

func (hi *ZapaHeightIndex) SetCheckpoint(blkID ids.ID) error {
	return database.PutID(hi.metadataDB, checkpointKey, blkID)
}

func (hi *ZapaHeightIndex) DeleteCheckpoint() error {
	return hi.metadataDB.Delete(checkpointKey)
}

func (hi *ZapaHeightIndex) DeleteBlockAtHeight(height uint64) error {
	key := database.PackUInt64(height)
	e := hi.heightDB.Delete(key)
	if e != nil {
		return e
	}
	return hi.DeleteCheckpoint()
}

