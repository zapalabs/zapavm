# zapavm

Avalanche is a network composed of multiple blockchains. Each blockchain is an instance of a [Virtual Machine (VM)](https://docs.avax.network/learn/platform-overview#virtual-machines), much like an object in an object-oriented language is an instance of a class. That is, the VM defines the behavior of the blockchain.

zapavm defines a vm that facilitates private transactions such that the identities of the sender and receiver are hidden. zapavm does this by utilizing an instance of `zcashd`. Note that this vm requires a corresponding running instance of `zcashd`. See [these instructions](https://github.com/rkass/zcash/blob/zsno-release/doc/debugging.md) for how to launch an instance of `zcashd` adopted for this use case. This go vm is a relay point between `rpcchainvm` and `zcashd`. Each block is a thin wrapper around a serialized zcash block. The go vm defined in this repo handles all networking and consensus and maintains minimal state, while the full chain is stored in the corresponding zcashd instance.

zapavm is served over RPC with [go-plugin](https://github.com/hashicorp/go-plugin).

# Building

- `./scripts/build.sh binaries zapavm`

# Testing

- You can use the launch.json defined [here](./.vscode/launch.json) to test out various zcash rpcs. This launch file invokes the main package with custom arguments that cause the script to run custom tests.