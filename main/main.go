// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-plugin"
	log "github.com/inconshreveable/log15"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/rpcchainvm"
	"github.com/zapalabs/zapavm/zapavm"
)

const logFile = "/avalanche-logs/zapa.log"

func main() {
	version, err := PrintVersion()
	if len(os.Args) == 2 {
		if os.Args[1] == "testBlockSerialization" {
			genesis := &zapavm.Block{
				PrntID: ids.Empty,
				Hght:   0,
				Tmstmp: time.Unix(0, 0).Unix(),
				ZBlk:   nil,
			}
			zc := zapavm.ZcashClient{}
			sugblk :=  zc.CallZcash("suggest", nil)
			block2 := &zapavm.Block{
				PrntID: genesis.ID(),
				Hght:   genesis.Height() + 1,
				Tmstmp: time.Now().Unix(),
				ZBlk:   sugblk.Result,
			}

			// Get the byte representation of the block
			block2Bytes, err := zapavm.Codec.Marshal(zapavm.CodecVersion, block2)
			if err != nil {
				return
			}

			newBlock := &zapavm.Block{}
			zapavm.Codec.Unmarshal(block2Bytes, newBlock)
			if newBlock.Height() != block2.Height() {
				panic("Discrepancy in height when unmarshalling")
			}
			if newBlock.Timestamp() != block2.Timestamp() {
				panic("Discrepancy in timestamp when unmarshalling")
			}
			if string(newBlock.ZBlock()[:]) != string(block2.ZBlock()[:]) {
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
		fmt.Printf("%s@%s\n", zapavm.Name, zapavm.Version)
		os.Exit(0)
	}

	fp, _ := filepath.Abs(logFile)
	lh, e := log.FileHandler(fp, log.TerminalFormat())
	if e != nil {
		fmt.Printf("Couldn't open log file handler %s", e)
		os.Exit(1)
	}
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlDebug, lh))
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: rpcchainvm.Handshake,
		Plugins: map[string]plugin.Plugin{
			"vm": rpcchainvm.New(&zapavm.VM{}),
		},

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
