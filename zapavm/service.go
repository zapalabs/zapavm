// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package zapavm

import (
	"errors"
	"fmt"
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

type NodeBlockCountRequest struct {
	FromHeight *int `json:"fromHeight,omitempty"` // inclusive
	ToHeight   *int `json:"toHeight,omitempty"` // exclusive
}

type GetBlockRequest struct {
	Height int `json:"height"`
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

type EnabledReply struct {
	Enabled bool
}

type NodeBlockCountReply struct {
	NodeBlockCounts map[string]int 
}

type BlockCountReply struct {
	Blocks int
}


// GetBlockArgs are the arguments to GetBlock
type GetBlockArgs struct {
	// ID of the block we're getting.
	// If left blank, gets the latest block
	ID *ids.ID `json:"id"`

	Height *int `json:"height"`
}

// GetBlockReply is the reply from GetBlock
type GetBlockReply struct {
	Timestamp json.Uint64 `json:"timestamp"` // Timestamp of most recent block
	Data      string      `json:"data"`      // Data in the most recent block. Base 58 repr. of 5 bytes.
	ID        ids.ID      `json:"id"`        // String repr. of ID of the most recent block
	ParentID  ids.ID      `json:"parentID"`  // String repr. of ID of the most recent block's parent
	ProducingNode string `json:"producingNode"`
}



func (s *Service) SubmitTx(_ *http.Request, args *SubmitTxArgs, reply *GetMempoolReply) error {
	log.Debug("SubmitTx: begin", "from", args.From, "to", args.To, "amount", args.Amount)
	result := s.vm.zc.SendMany(args.From, args.To, args.Amount)
	if result.Error != nil {
		s.vm.NotifyBlockReady()
		reply.SubmittedTx = result.Result
		s.vm.as.SendAppGossip(result.Result)
		reply.Mempool = nil
	}
	return result.Error
}

func (s *Service) GetBlockCount(_ *http.Request, args *EmptyArgs, reply *BlockCountReply) error {
	log.Debug("GetBlockCount: begin")
	b, e := s.vm.state.GetLastAcceptedBlock()
	if e != nil {
		return fmt.Errorf("Error fetching last accepted block %e", e)
	}
	reply.Blocks = int(b.Height())
	return nil
}

// tells the vm to mine a new block. will usually (but not 100%) cause this node to mine
func (s *Service) MineBlock(_ *http.Request, args *EmptyArgs, reply *SuccessReply) error {
	log.Debug("MineBlock: begin")
	if !TestNet {
		return errors.New("MineBlock can only be used on testnet and we are not on testnet")
	} 
	s.vm.NotifyBlockReady()
	reply.Success = true
	return nil
}

func (s *Service) Zcashrpc(_ *http.Request, args *zclient.ZCashRequest, reply *zclient.ZCashResponse) error {
	log.Debug("Zcashrpc: begin", "method", args.Method)
	result := s.vm.zc.CallZcashJson(args.Method, args.Params)
	reply.Result = result.Result
	reply.ID = result.ID
	reply.Error = result.Error
	return reply.Error
}

func (s *Service) IsChainEnabled(_ *http.Request, args *EmptyArgs, reply *EnabledReply) error {
	log.Debug("IsChainEnabled: begin", "nodeid", s.vm.ctx.NodeID)
	reply.Enabled = s.vm.enabled
	return nil
}

// associate with new zcash host and port
func (s *Service) AssociateZcashHostPort(_ *http.Request, args *ZcashHostInfo, reply *SuccessReply) error {
	log.Debug("AssociateZcashHostPort: begin", "rpc host", args.Host, "rpc port", args.Port)
	s.vm.zc.SetHost(args.Host)
	s.vm.zc.SetPort(args.Port)
	reply.Success = true
	return nil
}

func (s *Service) NodeBlockCounts(_ *http.Request, args *NodeBlockCountRequest, reply *NodeBlockCountReply) error {
	log.Debug("NodeBlockCounts: begin", "from height", args.FromHeight, "to height", args.ToHeight)
	reply.NodeBlockCounts = make(map[string]int)
	for blk := range s.vm.BlockGenerator(args.FromHeight, args.ToHeight) {
		if blk.ProducingNode != "" {		
			reply.NodeBlockCounts[blk.ProducingNode]++
		}
	}
	return nil
}

// GetBlock gets the block whose ID is [args.ID]
// If [args.ID] is empty, get the latest block
func (s *Service) GetBlock(_ *http.Request, args *GetBlockArgs, reply *GetBlockReply) error {
	// If an ID is given, parse its string representation to an ids.ID
	// If no ID is given, ID becomes the ID of last accepted block
	log.Debug("GetBlock: begin", "id", args.ID, "height", args.Height)
	var (
		id  ids.ID
		err error
		block *Block
	)


	if args.ID == nil {
		if args.Height == nil {
			return fmt.Errorf("must specify either height or id")
		}
		block, err = s.vm.GetBlockAtHeight(uint64(*args.Height))
	} else {
		id = *args.ID
		block, err = s.vm.getBlock(id)
	}

	// Get the block from the database
	
	if err != nil {
		return fmt.Errorf("Error retrieving block %e", err)
	}

	// Fill out the response with the block's data
	reply.ID = block.ID()
	reply.Timestamp = json.Uint64(block.Timestamp().Unix())
	reply.ParentID = block.Parent()
	reply.ProducingNode = block.ProducingNode
	reply.Data = string(block.Bytes())

	return err
}
