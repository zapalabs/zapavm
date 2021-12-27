// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package timestampvm

import (
	"errors"
	"net/http"
	log "github.com/inconshreveable/log15"
	nativejson "encoding/json"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/utils/json"
)

var (
	errBadData               = errors.New("data must be base 58 repr. of 32 bytes")
	errNoSuchBlock           = errors.New("couldn't get block from database. Does it exist?")
	errCannotGetLastAccepted = errors.New("problem getting last accepted")
)

// Service is the API service for this VM
type Service struct{ vm *VM }

// ProposeBlockArgs are the arguments to function ProposeValue
type ProposeBlockArgs struct {
	// Data in the block. Must be base 58 encoding of 32 bytes.
	Data string `json:"data"`
}

type SubmitTxArgs struct {
	Data nativejson.RawMessage `json:"data"`
}

type EmptyArgs struct {

}


// ProposeBlockReply is the reply from function ProposeBlock
type ProposeBlockReply struct{ Success bool }

type GetMempoolReply struct{ Mempool [][]byte
	 SubmittedTx []uint8 }

// ProposeBlock is an API method to propose a new block whose data is [args].Data.
// [args].Data must be a string repr. of a 32 byte array
func (s *Service) ProposeBlock(_ *http.Request, args *ProposeBlockArgs, reply *ProposeBlockReply) error {
	bytes, err := formatting.Decode(formatting.CB58, args.Data)
	if err != nil || len(bytes) != dataLen {
		return errBadData
	}
	var data [dataLen]byte         // The data as an array of bytes
	copy(data[:], bytes[:dataLen]) // Copy the bytes in dataSlice to data
	s.vm.proposeBlock(data)
	reply.Success = true
	return nil
}

func (s *Service) SubmitTx(_ *http.Request, args *SubmitTxArgs, reply *GetMempoolReply) error {

	var x []uint8 = []uint8{}
	for _, i := range(args.Data) {
		x = append(x, i)
	}
	log.Info("submitting transaction")
	reply.SubmittedTx = x
	s.vm.as.SendAppGossip(x)
	s.vm.mempool2 = append(s.vm.mempool2, x)
	reply.Mempool = s.vm.mempool2
	return nil
}

func (s *Service) GetMempool(_ *http.Request, args *EmptyArgs, reply *GetMempoolReply) error {
	log.Info("getting mempool")
	reply.Mempool = s.vm.mempool2
	return nil
}

// GetBlockArgs are the arguments to GetBlock
type GetBlockArgs struct {
	// ID of the block we're getting.
	// If left blank, gets the latest block
	ID *ids.ID `json:"id"`
}

// GetBlockReply is the reply from GetBlock
type GetBlockReply struct {
	Timestamp json.Uint64 `json:"timestamp"` // Timestamp of most recent block
	Data      string      `json:"data"`      // Data in the most recent block. Base 58 repr. of 5 bytes.
	ID        ids.ID      `json:"id"`        // String repr. of ID of the most recent block
	ParentID  ids.ID      `json:"parentID"`  // String repr. of ID of the most recent block's parent
}

// GetBlock gets the block whose ID is [args.ID]
// If [args.ID] is empty, get the latest block
func (s *Service) GetBlock(_ *http.Request, args *GetBlockArgs, reply *GetBlockReply) error {
	// If an ID is given, parse its string representation to an ids.ID
	// If no ID is given, ID becomes the ID of last accepted block
	var (
		id  ids.ID
		err error
	)

	if args.ID == nil {
		id, err = s.vm.state.GetLastAccepted()
		if err != nil {
			return errCannotGetLastAccepted
		}
	} else {
		id = *args.ID
	}

	// Get the block from the database
	block, err := s.vm.getBlock(id)
	if err != nil {
		return errNoSuchBlock
	}

	// Fill out the response with the block's data
	reply.ID = block.ID()
	reply.Timestamp = json.Uint64(block.Timestamp().Unix())
	reply.ParentID = block.Parent()
	data := block.Data()
	reply.Data, err = formatting.EncodeWithChecksum(formatting.CB58, data[:])

	return err
}
