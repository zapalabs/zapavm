// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package zapavm

import (
	"errors"
	"net/http"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/json"
	log "github.com/inconshreveable/log15"
	"github.com/zapalabs/zapavm/zapavm/zclient"
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
	From   string  `json:"from"`
	To     string  `json:"to"`
	Amount float32 `json:"amount"`
}

type EmptyArgs struct {
}

type ZcashHostInfo struct {
	Host   string `json:"host"`
	Port   int `json:"port"`
}

// ProposeBlockReply is the reply from function ProposeBlock
type ProposeBlockReply struct{ Success bool }

type GetMempoolReply struct {
	Mempool     [][]byte
	SubmittedTx []uint8
}

type SuccessReply struct {
	Success bool
}

func (s *Service) SubmitTx(_ *http.Request, args *SubmitTxArgs, reply *GetMempoolReply) error {
	log.Info("submitting transaction. calling zcash.zendmany from", "nodeid", s.vm.ctx.NodeID)
	result := s.vm.zc.SendMany(args.From, args.To, args.Amount)
	s.vm.NotifyBlockReady()
	reply.SubmittedTx = result.Result
	s.vm.as.SendAppGossip(result.Result)
	reply.Mempool = nil
	return nil
}

// tells the vm to mine a new block. will usually (but not 100%) cause this node to mine a block
func (s *Service) MineBlock(_ *http.Request, args *SubmitTxArgs, reply *SuccessReply) error {
	if !TestNet {
		return errors.New("MineBlock can only be used on testnet and we are not on testnet")
	} 
	log.Info("suggesting empty block to reap coinbase rewards. can only be used on testnet", "nodeid", s.vm.ctx.NodeID)
	s.vm.NotifyBlockReady()
	reply.Success = true
	return nil
}

func (s *Service) Zcashrpc(_ *http.Request, args *zclient.ZCashRequest, reply *zclient.ZCashResponse) error {
	log.Info("calling zcash rpc", "nodeid", s.vm.ctx.NodeID)
	result := s.vm.zc.CallZcashJson(args.Method, args.Params)
	reply.Result = result.Result
	reply.ID = result.ID
	reply.Error = result.Error
	return nil
}

// needed to associate with local zcash rpc when multiple are running on same machine
func (s *Service) AssociateZcashHostPort(_ *http.Request, args *ZcashHostInfo, reply *SuccessReply) error {
	log.Info("calling associate zcash host port", "rpc host", args.Host, "rpc port", args.Port)
	s.vm.zc.SetHost(args.Host)
	s.vm.zc.SetPort(args.Port)
	reply.Success = true
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

	return err
}
