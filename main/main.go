// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/hashicorp/go-plugin"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/rpcchainvm"
	"github.com/zapalabs/zapavm/zapavm"
	"github.com/zapalabs/zapavm/zapavm/zclient"
)

func main() {
	version, err := PrintVersion()
	if len(os.Args) == 2 {
		if os.Args[1] == "testBlockSerialization" {
			genesis := &zapavm.Block{
				PrntID: ids.Empty,
				Hght:   0,
				ZBlk:   nil,
			}
			zc := &zclient.ZcashHTTPClient{}
			sugblk := zc.CallZcash("suggest", nil)
			block2 := &zapavm.Block{
				PrntID: genesis.ID(),
				Hght:   genesis.Height() + 1,
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
		if os.Args[1] == "iterateBlocks" {
			zc := &zclient.ZcashHTTPClient{}
			zc.Port = 8233
			x := 0
			for i := range(zclient.BlockGenerator(zc)) {
				fmt.Print(i.Block)
				fmt.Print(i.Timestamp)
				x++;
			}
			fmt.Print(x)
			return
		}
		if os.Args[1] == "testLaunchScript" {
			cmd, err := exec.Command("/bin/sh", "/Users/rkass/repos/zapa/zapavm/main/script.sh").Output()
			if err != nil {
				fmt.Printf("error %s", err)
			}
			output := string(cmd)
			fmt.Print(output)
		}
		if os.Args[1] == "generatevmid" {
			// doesn't work
			id,err := ids.FromString("zapavm")
			if err != nil {
				fmt.Printf("error %s", err)
				os.Exit(1)
			}
			fmt.Print(id)
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

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: rpcchainvm.Handshake,
		Plugins: map[string]plugin.Plugin{
			"vm": rpcchainvm.New(&zapavm.VM{}),
		},

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
