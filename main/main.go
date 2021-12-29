// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-plugin"
	log "github.com/inconshreveable/log15"

	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/vms/rpcchainvm"
	"github.com/ava-labs/timestampvm/timestampvm"
)

const GenesisBlockFile = "../builds/genesis.txt"
const logFile = "../timestampvm2/logs/log"

const GenesisBlockString = "HaVC\x0e\xf7F<j\xc4\x10\tw\x90\xe8.k-xk\xf51\xc0+ehS\xb9A\xfe\xa1%"

func main() {
	version, err := PrintVersion()
	if len(os.Args) == 2 {
		if os.Args[1] == "getGenesis" {
			block := &timestampvm.Block{
				Hght:   0,
				Tmstmp: time.Now().Unix(),
				ZBlk: nil,
			}
		
			// Get the byte representation of the block
			blockBytes, err := timestampvm.Codec.Marshal(timestampvm.CodecVersion, block)
			if err != nil {
				return 
			}

			block.Initialize(blockBytes, choices.Accepted, nil)

			fp, _ := filepath.Abs(GenesisBlockFile)
			ioutil.WriteFile(fp, blockBytes, 777)
			log.Info("Wrote genesis block to ", fp)
			return
		} else if os.Args[1] == "testBlockSerialization" {
			fp, _ := filepath.Abs(GenesisBlockFile)
			blockBytes, _ := ioutil.ReadFile(fp)
			block := &timestampvm.Block{}
			timestampvm.Codec.Unmarshal(blockBytes, block)
			block.Initialize(blockBytes, choices.Accepted, nil)
			if (block.ID().String() == GenesisBlockString) {
				panic("Unexpected genesis block id")
			}
			if (block.ZBlock() != nil) {
				panic("Genesis block should have nil ZBlock")
			}
			// on node 2 sugest a block and return it
			sugblk := timestampvm.CallZcash("suggest", nil, 2)
			block2 := &timestampvm.Block{
				PrntID: block.ID(),
				Hght:   block.Height() + 1,
				Tmstmp: time.Now().Unix(),
				ZBlk: sugblk.Result,
			}
		
			// Get the byte representation of the block
			block2Bytes, err := timestampvm.Codec.Marshal(timestampvm.CodecVersion, block2)
			if err != nil {
				return
			}

			newBlock := &timestampvm.Block{}
			timestampvm.Codec.Unmarshal(block2Bytes, newBlock)
			if (newBlock.Height() != block2.Height()) {
				panic("Discrepancy in height when unmarshalling")
			}
			if (newBlock.Timestamp() != block2.Timestamp()) {
				panic("Discrepancy in timestamp when unmarshalling")
			}
			if (string(newBlock.ZBlock()[:]) != string(block2.ZBlock()[:])) {
				panic("Discrepancy in zblock when unmarshalling")
			}
			
			return
		}
	}

	if err != nil {
		fmt.Printf("couldn't get config: %s", err)
		os.Exit(1)
	}
	// Print VM ID and exit
	if version {
		fmt.Printf("%s@%s\n", timestampvm.Name, timestampvm.Version)
		os.Exit(0)
	}

	fp, _ := filepath.Abs(logFile)
	lh, e := log.FileHandler(fp, log.TerminalFormat())
	if e != nil {
		fmt.Printf("Couldn't open log file handler %s", e)
		os.Exit(1)
	}
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlDebug, lh))
	// log.Root().SetHandler(log.LvlFilterHandler(log.LvlDebug, log.StreamHandler(os.Stderr, log.TerminalFormat())))
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: rpcchainvm.Handshake,
		Plugins: map[string]plugin.Plugin{
			"vm": rpcchainvm.New(&timestampvm.VM{}),
		},

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
