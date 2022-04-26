// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package zapavm

import (
	"errors"
	"time"

	nativejson "encoding/json"

	log "github.com/inconshreveable/log15"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/utils/hashing"
)

var (
	errTimestampTooEarly = errors.New("block's timestamp is earlier than its parent's timestamp")
	errDatabaseGet       = errors.New("error while retrieving data from database")
	errTimestampTooLate  = errors.New("block's timestamp is more than 1 hour ahead of local time")

	_ snowman.Block = &Block{}
)

// Block is a block on the chain.
// Each block contains:
// 1) ParentID
// 2) Height
// 3) ZBlk -- the serialized zcash block
type Block struct {
	PrntID ids.ID                `serialize:"true" json:"parentID"`  // parent's ID
	Hght   uint64                `serialize:"true" json:"height"`    // This block's height. The genesis block is at height 0.
	ZBlk   nativejson.RawMessage `serialize:"true" json:"zblock"`    // zcash block
	CreationTime int64              `serialize:"true" json:"creationTime"`

	id     ids.ID         // hold this block's ID
	bytes  []byte         // this block's encoded bytes
	status choices.Status // block's status
	vm     *VM            // the underlying VM reference, mostly used for state
}

// Verify returns nil iff this block is valid.
func (b *Block) Verify() error {
	log.Info("Calling verify block", "nodeid", b.vm.ctx.NodeID.String(), "height", b.Height())
	if b.ZBlock() != nil {
		err := b.vm.zc.ValidateBlock(b.ZBlock()) 
		if err != nil {
			log.Warn("Validate block returned with an error", "error", err)
			return err
		}
	}

	log.Info("Successfully validated block")
	b.vm.verifiedBlocks[b.ID()] = b

	return nil
}

// Initialize sets [b.bytes] to [bytes], [b.id] to hash([b.bytes]),
// [b.status] to [status] and [b.vm] to [vm]
func (b *Block) Initialize(bytes []byte, status choices.Status, vm *VM) {
	b.bytes = bytes
	b.id = hashing.ComputeHash256Array(b.bytes)
	b.status = status
	b.vm = vm
}

// Accept sets this block's status to Accepted and sets lastAccepted to this
// block's ID and saves this info to b.vm.DB
func (b *Block) Accept() error {
	log.Info("Calling accept block", "nodeid", b.vm.ctx.NodeID.String(), "height", b.Height())

	if b.Height() > 0 {
		// Needs to be synced with Zcash Client
		log.Info("Calling zcash submit block", "nodeid", b.vm.ctx.NodeID.String(), "height", b.Height())
		b.vm.zc.SubmitBlock(b.ZBlock())
	}

	b.SetStatus(choices.Accepted) // Change state of this block
	blkID := b.ID()

	// Persist data
	if err := b.vm.state.PutBlock(b); err != nil {
		return err
	}

	// Set last accepted ID to this block ID
	if err := b.vm.state.SetLastAccepted(blkID); err != nil {
		return err
	}

	// Delete this block from verified blocks as it's accepted
	delete(b.vm.verifiedBlocks, b.ID())

	// Commit changes to database
	return b.vm.state.Commit()
}

// Reject sets this block's status to Rejected and saves the status in state
// Recall that b.vm.DB.Commit() must be called to persist to the DB
func (b *Block) Reject() error {
	log.Info("Calling reject block", "nodeid", b.vm.ctx.NodeID.String(), "height", b.Height())

	b.SetStatus(choices.Rejected) // Change state of this block
	if err := b.vm.state.PutBlock(b); err != nil {
		return err
	}
	// Delete this block from verified blocks as it's rejected
	delete(b.vm.verifiedBlocks, b.ID())
	// Commit changes to database
	return b.vm.state.Commit()
}

// ID returns the ID of this block
func (b *Block) ID() ids.ID { return b.id }

// ParentID returns [b]'s parent's ID
func (b *Block) Parent() ids.ID { return b.PrntID }

// Height returns this block's height. The genesis block has height 0.
func (b *Block) Height() uint64 { return b.Hght }

// Timestamp returns this block's time. The genesis block has time 0. For now, return
// the root timesamp (2022-01-01) for genesis plus one second for each additional block
func (b *Block) Timestamp() time.Time { 
	return time.Unix(b.CreationTime, 0)
}

// Status returns the status of this block
func (b *Block) Status() choices.Status { return b.status }

// Bytes returns the byte repr. of this block
func (b *Block) Bytes() []byte { return b.bytes }

func (b *Block) ZBlock() nativejson.RawMessage {
	return b.ZBlk
}

// SetStatus sets the status of this block
func (b *Block) SetStatus(status choices.Status) { b.status = status }
