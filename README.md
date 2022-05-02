# Zapavm

Avalanche is a network composed of multiple blockchains. Each blockchain is an instance of a [Virtual Machine (VM)](https://docs.avax.network/learn/platform-overview#virtual-machines), much like an object in an object-oriented language is an instance of a class. That is, the VM defines the behavior of the blockchain.

Zapavm defines a virtual machine that facilitates private transactions such that the identities of the sender and receiver are hidden. Zapavm does this by utilizing an instance of `zcashd`. Note that this vm requires a corresponding running instance of `zcashd`. See [these instructions](https://github.com/zapalabs/zcash/blob/master/doc/running.md) for how to launch an instance of `zcashd` adopted for this use case. This go vm is a relay point between `rpcchainvm` and `zcashd`. Each block is a thin wrapper around a serialized zcash block. The go vm defined in this repo handles all networking and consensus.

Zapavm is served over RPC with [go-plugin](https://github.com/hashicorp/go-plugin).

# Builds

This repo comes with two pre-built binaries [zapavm-ubuntu](./builds/zapavm-ubuntu) and [zapavm-osx](./builds/zapavm-osx)

# Building

`./scripts/build.sh builds zapavm`

# Testing

- You can use the launch.json defined [here](./.vscode/launch.json) to test out various zcash rpcs. This launch file invokes the main package with custom arguments that cause the script to run custom tests.

# API

The Zapavm defines RPC endpoints for interacting with the blockchain. Some of these endpoints direct Zapavm to forward a request to the [Zcash API](https://github.com/zapalabs/zcash/blob/master/doc/api.md).

## Methods

### zapavm.zcashrpc

In order to provide maximum flexibility, this RPC method acts as a pass through to [Zcash API](https://github.com/zapalabs/zcash/blob/master/doc/api.md), allowing the user to issue any API call defined there via this method. Listed below are examples for each endpoint that has been thus far useful.

#### Arguments

```
{
   `"params" string[]`      A list of params that get forwarded to Zcash API.
   `"Method" string`        The Zcash API method to call.
   `"ID"     string`        ID for this request.
}
```

##### Result

Method specific.

#### Example: List Unspent

For each example, see the Zcash API spec for details about the parameters and return values.

##### Request

```
curl --location --request POST 'http://$HOST:$PORT/ext/bc/$BLOCKCHAIN' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "zapavm.zcashrpc",
    "params":{
        "params":[],
        "Method":"listunspent",
        "ID": "fromavax"
    },
    "id": 1
}
'
```
##### Response
```
{
    "jsonrpc": "2.0",
    "result": {
        "result": [
            {
                "txid": "eb9bef510aba0985374774f3c8d9e91218c4336012bedafbce7d6bf1e05010f6",
                "vout": 0,
                "generated": true,
                "address": "tmBtYWLLmZQL8ffpNBUDbpUR1uqmpeJZUKz",
                "scriptPubKey": "76a91417ddbd62bfde68650cc89f72e599cf9ed231d1b988ac",
                "amount": 12.50000000,
                "amountZat": 1250000000,
                "confirmations": 49,
                "spendable": true
            },
            {
                "txid": "eb7c9ed1fc9b8907952399fc9a1486e9bab3bcffcc8c27bac8dcc013d4d86efe",
                "vout": 0,
                "generated": true,
                "address": "tmM4uykd8q77Ptj9Xphk8BZkSDCwWyJ4ARE",
                "scriptPubKey": "76a9147c8cf4bc35ad6754fbc8beabc9488f1c370018e288ac",
                "amount": 12.50000000,
                "amountZat": 1250000000,
                "confirmations": 4,
                "spendable": true
            }
        ],
        "error": null,
        "id": "fromgo"
    },
    "id": 1
}
```

#### Example: getblockcount

##### Request

curl --location --request POST 'http://$HOST:$PORT/ext/bc/$BLOCKCHAIN' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "zapavm.zcashrpc",
    "params":{
        "params":[],
        "Method":"getblockcount",
        "ID": "fromavax"
    },
    "id": 1
}
'

##### Response

```
{
    "jsonrpc": "2.0",
    "result": {
        "result": 124,
        "error": null,
        "id": "fromgo"
    },
    "id": 1
}
```

#### Example: z_getbalance

##### Request

```
curl --location --request POST 'http://$HOST:$PORT/ext/bc/$BLOCKCHAIN' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "zapavm.zcashrpc",
    "params":{
        "params":["zregtestsapling1qx3m2j2z58828q5zusg9xt2x9j894wucaaljvwy58t5l4u9wzqf8zwdjm04ugh77d7svcp6cfft"],
        "Method":"z_getbalance",
        "ID": "fromavax"
    },
    "id": 1
}
'
```

##### Response

```
{
    "jsonrpc": "2.0",
    "result": {
        "result": 1.00000000,
        "error": null,
        "id": "fromgo"
    },
    "id": 1
}
```

#### Example: z_getnewaddress

##### Request

```
curl --location --request POST 'http://$HOST:$PORT/ext/bc/$BLOCKCHAIN' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "zapavm.zcashrpc",
    "params":{
        "params":[],
        "Method":"z_getnewaddress",
        "ID": "fromavax"
    },
    "id": 1
}
'
```

##### Response

```
{
    "jsonrpc": "2.0",
    "result": {
        "result": "zregtestsapling1qx3m2j2z58828q5zusg9xt2x9j894wucaaljvwy58t5l4u9wzqf8zwdjm04ugh77d7svckhacer",
        "error": null,
        "id": "fromgo"
    },
    "id": 1
}
```

### zapavm.mineBlock

Instruct this node to mine a block (may be empty). This method is not allowed in production, however it's used in fuji in order to generate coinbase rewards for validators so that validators can gain experience sending and receiving ZAPA.

#### Result
```
{
  `"Success" boolean` indicating whether or not this call succeeded. A true value here does not necessarily mean that this node proposed a block which made it into consensus, however a true value here indicates this node proposed a block to consensus.
}
```

#### Example

##### Request

curl --location --request POST 'http://$HOST:$PORT/ext/bc/$BLOCKCHAIN' \--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "zapavm.mineBlock",
    "params":{
    },
    "id": 1
}
'

##### Response

```
{
    "jsonrpc": "2.0",
    "result": {
        "Success": true
    },
    "id": 1
}
```

### zapavm.getBlockCount

Get the number of blocks that have been accepted into the chain.

#### Result

```
{
  `"Blocks" integer` Current block height (how many non-genesis blocks have been produced).
}
```

#### Example

##### Request

```
curl --location --request POST 'http://$HOST:$PORT/ext/bc/$BLOCKCHAIN' \--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "zapavm.getBlockCount",
    "params":{
    },
    "id": 1
}
'
```

##### Response

```
{
    "jsonrpc": "2.0",
    "result": {
        "Blocks": 115
    },
    "id": 1
}
```

### zapavm.getBlockAtHeight

Get information about the block at the specified height

#### Arguments
```
{
  `"Height" integer` indicates which block for which the client is requesting information.
}
```

#### Result

```
{
  `"timestamp"    integer`      Time the block was produced.
  `"data"         string`       Serialized byte representation of the block.
  `"id"           string`       Block identifier.
  `"parentID      string`       Block identifier of this block's parent.
  `"producingNode string`       NodeID of the validator which produced this block.
}
```

#### Example

##### Request

```
curl --location --request POST 'http://$HOST:$PORT/ext/bc/$BLOCKCHAIN' \--header 'Content-Type: application/json' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "zapavm.getBlock",
    "params":{
        "Height": 115
    },
    "id": 1
}
'
```


##### Response

```
{
    "jsonrpc": "2.0",
    "result": {
        "timestamp": "1651108718",
        "data": "asadsa[a,0,0,60,94,217,13,193,79,51,246,49,251,194,130,189,106,4,75,129,30,244,94,104,32,119,101,204,250,187,61,109,229,42,163,251,109,56,250,199,220,88,184,34,21,163,29,114,5,87,163,92,5,222,170,4,99,126,190,198,134,66,115,50,151,203,172,61,35,82,227,67,94,132,137,183,231,77,218,137,247,56,180,11,101,196,152,173,101,155,16,232,197,135,250,156,3,3,250,110,235,105,98,15,15,15,32,1,0,229,118,39,230,12,93,248,80,108,58,24,227,91,61,233,161,233,201,186,149,151,101,168,73,83,116,35,136,0,0,0,1,5,0,0,128,10,39,167,38,33,150,81,55,0,0,0,0,115,0,0,0,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,255,255,255,255,4,1,115,1,1,255,255,255,255,1,128,124,129,74,0,0,0,0,25,118,169,20,240,101,157,146,209,64,25,191,70,183,232,42,39,159,190,235,151,198,231,179,136,172,0,0,0]YHcLqA1DCabUos5uQUJmr98cG3aWBsQ",
        "id": "2mZTJVKK6vXb7cGxhcN8viDuUxwsmePCrhfyfbEFyt61uBbPhU",
        "parentID": "2fpMEGoSsxHV6VLCARCGuXTkPa6UvWduxPJLSvmyozRmDjUeeb",
        "producingNode": "FfYHcLqA1DCabUos5uQUJmr98cG3aWBsQ"
    },
    "id": 1
}
```

### zapavm.submitTx

Submits a transaction to the blockchain. 

#### Arguments
```
{
  `"from"   string` Address sending funds. This node's zcash must own this address.
  `"to"     string` Address receiving funds.
  `"amount" int`    How much ZAPA to transfer.
}
```

#### Result

```
{
  `"Mempool"      null [deprecated]`
  `"SubmittedTx"  string`               Serialized byte representation of the transaction.
}
```

#### Example

##### Request

```
curl --location --request POST 'http://$HOST:$PORT/ext/bc/$BLOCKCHAIN' \--header 'Content-Type: application/json' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "zapavm.submitTx",
    "params":{
        "from":"zregtestsapling1rq3epttsc74ehydcx7a9tx4wklerrjly0qlm8hzglmqgsz5a2m0sfanvz0rcxkv4c23lyvec4wj",
        "to":"zregtestsapling1qx3m2j2z58828q5zusg9xt2x9j894wucaaljvwy58t5l4u9wzqf8zwdjm04ugh77d7svcp6cfft",
        "amount": 1
    },
    "id": 1
}
'
```


##### Response

```
{
    "jsonrpc": "2.0",
    "result": {
        "Mempool": null,
        "SubmittedTx": "WzUsMCwwLDEyOCwxMCwzOSwxNjcsMzgsMzMsMTUwLDgxLDU1LDAsMCwwLDAsMTQzLDAsMCwwLDAsMCwxLDEyNyw2Nyw4OSwxOSwxNTksMjQ1LDEwMiwyMTIsMTIyLDIwNSwyMjcsMTkyLDc0LDE0MSwxODEsMTg5LDE0MSw5NiwyNiwxMTEsMzgsODMsODUsMTk1LDIzOCwyNTAsMzYsNDcsMTQ3LDQwLDE2NCw2MCwxMTMsNDcsNywxMzEsMSwxNzQsMjAsMjE3LDMyLDc3LDE0LDE2MCwxMzYsNjUsMiwyMDIsMTYsMTQ3LDIzMywxNTksMTg5LDMxLDIzMiwxODEsMTE1LDc2LDE1MSw0NCwyOCw3MCw5MiwxNTMsNjQsMTY1LDIxMyw0NCwxNjQsOTgsMTk2LDEsNzEsMjQzLDE0LDIyMSwyNTMsMTgxLDIxLDI0MiwyMjksMjE1LDE2OCw3MiwyMCw2NCwyMDksMjA4LDczLDE4LDcsNjEsMjAxLDEyNCwyMTcsMTYzLDIsMTIwLDQ2LDE5NSwyMDksMzEsMjM1LDE5NSwyNCw4MywxODQsMTE0LDEyOCwxMSw5OCwzMywyNSwxOCwyNDEsNDgsMTYxLDE1MCwxNiw2LDkxLDY3LDI5LDExOSwyMzAsMTcyLDIyNCwyNCwyMCwyNDUsMzgsMjAsODMsMTg0LDE2NCwyMDQsMTQwLDEsMTUwLDc5LDYwLDEzNSwxOTgsMTY0LDE3MCw1NSw0OSw2MywxNDMsMjE5LDM2LDIwOSwxNDAsMTY2LDYsNjcsMTIsMTc4LDE4MiwxODAsODgsMjUyLDIyMyw1LDUwLDE2NCwxODMsMTUsMTI5LDIzNSwxODksMzAsMzIsMTk4LDEzNSw2MCwxMDUsOTUsODksMCwyNDUsMTA2LDE2OSw1NiwxNjgsMzAsMTg5LDEzMSw1Myw5NiwxMjcsMTEzLDEwNiwyMjQsMTMzLDkwLDI0LDM2LDYsMjI4LDY5LDIxMCwxNjcsMTMxLDI1MSwzOSwxNDIsMTQ5LDc2LDIwMSwxNjksMjA3LDMzLDEzOSwxMjgsMTcwLDc2LDE3NSwxMjYsOTgsMjI4LDExOSwyNiwyMDQsMTg2LDk2LDM0LDI1MCw4Niw0NCwyMTksMTkwLDE4NiwyMDMsMTksMjA1LDExMywyMTEsMTM4LDIyOSw0MSwxMzQsMjIxLDEyMSwyMiwxMzQsMTE5LDExNiw0NiwxMCwyMTksMjUzLDIxLDE5NCwyMjgsMjExLDEwNywxMjQsMjQwLDE3NywyMiwxMzYsMjQwLDE2Myw2MywyMTAsMzgsMTI5LDY1LDIzNiwxNzYsMTcsMjcsMTgzLDYzLDIwNSwyMjAsMTcxLDE5MSwyMjYsMTYwLDgwLDgyLDEwOCwzNywxNTMsMTU3LDE3MiwxNjEsMjUxLDkyLDE0OSwxMTksMTE0LDI5LDIzOCwxMzUsMTAsMTY2LDE2LDE3NywxNDQsMjE5LDUzLDY5LDE1NywzNCw5MiwyMDUsMjQxLDEzNCwzNyw2MSwxMTgsMTU0LDAsMjMzLDYwLDQ1LDE0NCwxMTQsMjE2LDE3NCwzMCwxMDcsMjUxLDcyLDE0NywxMzcsMTkzLDExOSwyMjQsMjE3LDExNiwzNSw3NSwyMjUsMjMwLDgyLDI2LDg4LDMyLDIzMCwxNDYsMjQyLDEzNiwyMjEsMjEyLDI1MCwzNyw0NiwxMDQsMTcwLDIzMiwyMDgsMTY0LDI0NiwyMDMsMTkyLDYsMTUsMTk3LDE4NiwxOTYsMTA2LDExOCwxMTcsMTYxLDEyNiwyNTEsMTA2LDEyMSw1NiwxMDAsMTc4LDE1MiwxNjksMTQ1LDI1MSwyMywxMzcsMjA2LDE0OCw1NiwzNyw3MiwyMTEsMTI3LDI0MSw3NCwyMzgsNjksMTIyLDk1LDE0OSw0Myw4OCwxMzEsMjUsMTc5LDE0OSw5OCw2Myw4Myw2NiwxNjYsMTcxLDIxMCw0OSw0MCwyMzIsMTcsMTc1LDExOCwxMzEsNzUsOTYsMjQ0LDEyMywyMTQsMTk4LDIzNSw4LDIzNyw3NCw0NSw5LDE5Nyw0Myw3MCwxNDIsNjEsMjMyLDIzNiwxOTAsMTM2LDM1LDIyMCwxNTAsMTU0LDEzLDEzMSwxMTMsMjYsMjAsMjUyLDQxLDI0LDcxLDE2Niw4NSwyNywxNzMsMTc4LDE3NCwxNDQsMiwxMDMsNywyMSwxNzksMTY5LDE3NywyMCwxNDEsMTExLDk3LDUsMjA4LDEwNywxNDYsMTYwLDE5Miw2NCwxNTQsMTMyLDIyOSwyNDEsMTgwLDE2OCw4MywxMjUsMjE3LDEzOCw4Miw1MiwxMjIsMTY0LDEwOSw4MSwxNzcsMTM2LDEwNSwxNDMsMjQ0LDEzMSwxNDgsNywxNTgsMjQxLDIyNiwxMTYsOTMsNzMsMTYzLDEwMSw4MywxNTYsMTIzLDM3LDE1OSwxNjcsMjM1LDU2LDEwNyw3MSwyMjYsMTg2LDE1LDI0LDUyLDE4LDIyNiw1Miw4MSwyNywzNiwxODMsOTcsMjIyLDU1LDEwOSwyMjAsMTAsMTU1LDMxLDkyLDksMTE3LDk0LDcxLDEyMCwxNzcsMjAzLDI4LDIxNCwxOCwxNzIsMjE1LDE4OSwxOTYsMTkwLDI1NCwxNzUsMTYxLDgzLDE1LDIwOSw1Miw3NCwxOTAsNzIsNzQsMTEzLDE4MSw2MSw4NywxNDcsODAsMjMzLDI0MiwxOTUsMTMwLDEzNCw5MiwyMjYsMjEzLDEwMSwyMCw4OSwxLDg2LDIwNCw5LDIyMSwxMTYsNzQsMjUwLDM4LDg1LDEzMCw3LDE4OSwxNDEsNTgsMTM5LDIyNCwxNTIsODIsMTc4LDksMjYsMTMxLDc3LDI2LDMyLDEwLDE0LDQ4LDE4NSwxMTQsMTM4LDk2LDIzMCw3MiwxODYsMjksODYsMTg1LDE5MCwwLDIyNiw5MCwxNzYsMTA1LDE3NCw2MywxODgsMTYyLDI1MSwxNzYsOTEsMCwyLDEwLDM4LDE4OSwxNjEsMTAsOTQsMTc5LDU3LDE4NywyNDAsMyw3MywyMzEsMjI1LDE5NSwxNTQsMTk0LDE0MCw2NCwxNjYsMjI4LDc0LDExMCwxODgsMTc1LDExOCwxMjMsNDUsMTM3LDE4NywxMjYsMjUxLDE0OSwxNTAsMjUsMTcwLDc4LDEyLDEzMCwxMzMsMTM5LDMyLDIwLDEyMSwyNDMsMjUyLDI0NiwxODAsMTA1LDI0OCw0MSwyMzAsNjAsMTIsMTQ2LDIzLDIyMyw3NCwxNDgsMTg0LDgxLDEwNSw0MCwyMTEsMTcwLDMzLDEzMiwxNTMsMTEwLDE2OCwxMTMsMTQ1LDMxLDIyNiwxNTYsOTUsMTk4LDEzMSwxNzEsMTcxLDQwLDgsMTg3LDY3LDc3LDIwNiwxMTksOCwxODcsMTM3LDIzOSwxMzAsMTgyLDE5MSwxMDQsNDksMTc5LDEwNiwxMDksMTk4LDE2NywyNSwxMjcsMzUsMTAsMjMxLDE4Miw5MSwxOTEsMTg5LDI0NSwxMjEsNDcsNDUsMjA4LDQ5LDE4OCwyMjgsMjI4LDE2LDIxLDEwLDgsMTg1LDE3MywxNTYsNjUsMTk4LDEzMSwxOTgsMTY5LDIzNiwyMzMsMTYyLDIxNCwyNDEsMTI4LDY0LDksNzUsMjQ2LDIzOCwyMDUsNzcsMjAsMjEzLDg1LDIyLDEzMiwxNTcsMTU0LDE5NywyMTgsMTk5LDE1NiwyMjMsMzQsMTI2LDI0NywxOTEsMTQsMTk5LDgzLDk3LDE1OSwxNjgsMjEsNDIsMjExLDI4LDIyLDEsMjAxLDc5LDE3Miw4OSwzNywxMjQsMTM0LDEyMywxMTIsMTIsMTk2LDE1NywxNSw3LDIsMTA0LDk2LDIwMCwyMTUsMjI3LDE5MCwxNTEsMjUsNDEsMTUzLDYzLDQxLDg4LDE3MywxMTcsMTgxLDIzNiw1MCwyMjksMzcsMTY4LDE4MSwxNjgsMTg0LDUyLDExMiw0LDE2OSwyMTcsMjQ2LDEzNSwxNzAsNzUsMTExLDMzLDE0NywyNDIsMTQ4LDEwMCwxMzgsMTM1LDIzNywxMjEsMzgsMTQ4LDIxMywxNTMsNzQsNjYsNjMsMTU3LDg3LDEyLDg2LDE4NSwxOTEsMjEwLDEyMywxNzQsNzEsMTkzLDIxOSwyMzIsNzIsMTgxLDkzLDEyLDIzNCwyOSw0MSwzMSwxNzYsNTAsMjA0LDk2LDYsMTI0LDIyNiwyMTcsMjIyLDE5MSw4MCw4MiwxMjQsMjM2LDE1OSwyNyw5OCw0NCwxNjMsMTMyLDEyMiwyMDMsMjA0LDE5OCw4MiwxNDQsMTA2LDEyNywyMjQsMTQ5LDE0LDEzMywxODMsNTksMzYsMjQyLDgsMzIsMjIsMjM2LDEzNiwxNjksMTQyLDI0LDE0MiwxODIsMTQzLDc0LDExNywxOTEsMTk5LDU3LDIwLDIyMywxNzYsNjMsNjEsMTcwLDIxMiwyMzEsMTIxLDUsMTksMTQ5LDM4LDQ4LDIzMiw1MiwyMjYsNzMsMjQsMTAsNzksMjMzLDIzNywxMzAsMTYxLDcyLDExMywyMDYsMTc5LDI0OCwyMDgsODEsMTc3LDE1NywxMzIsMTYwLDE3MiwxMjUsNDYsMTYwLDE4OSwxOTEsMjM0LDI0OSwxNzcsMTc3LDE4NSwxMTcsMTcyLDg2LDI4LDY3LDE4MSwxMCwxNTMsMTk1LDMsMjM0LDc5LDE0MCwzNiwyNTAsMjQ1LDQyLDI1Myw1MCwyMTQsMjM4LDIyNywxNTEsMjE4LDEyNiwxMDIsMjA1LDI1NCwyNDQsOTksMjIxLDM1LDc3LDg1LDIxOSwyNDcsMjAyLDU1LDE0NSwxMTYsOTYsMTYwLDMzLDgyLDEzMSw4MiwzOSw2NSwxOTAsMjQsOTIsMTc5LDgsMjA4LDExMCw3NCw3MywxODgsMjUsMTQ4LDExNiw2NiwyMTcsMjEsMjEzLDExNyw0NywxODcsNDMsMjI1LDY1LDEwNCw0OSwyNDcsMSw3NSwyMDYsMTM5LDEwLDEzOSwxOTEsMTgyLDkzLDE4NiwxMDksNjQsMTk1LDIwNyw1Myw2LDE5NywyMDYsMSwxMTEsMjMwLDI1MSwyNSwxNjEsMjA4LDIxNSwyNDYsNjEsMjExLDE0MCwxNzUsMTg3LDE1MSw3NSw2MSwxNzUsMjAsMTQyLDE5NCwxOTYsMjIzLDIzNCwxNjcsMjMzLDE3MCwxNDYsMCwxNSwxMzYsNzcsMjQ4LDIxNSwxNjQsODUsMTU2LDE4MiwxMDYsMTE5LDI0MiwxNzMsOTUsNTMsMjQ3LDE3OCwyMTcsNjQsNTcsODMsMjI3LDExNSwxOTEsMTMzLDIzMSw1OSwxMTQsNDcsMTk3LDgwLDE2Niw0OSwzNCwxMjAsMjM2LDIzNSwxOCwxOTEsNCwxMiwxMDksMjUsMzAsMTU2LDI1NCwyNCwxOCwxNDksMjQ1LDEyMiwxODEsMTU3LDI0MywxNTUsMjQ1LDEyNSwxNzEsNDgsMjM4LDIwNCwyMzYsMTk4LDIxMiwxMCwyMiwxODIsMjQxLDEzNSwzMiwxMSwyMDgsODgsMjQsMTEyLDI0MywxNDYsMTgzLDIyMiw4Myw1MSw2NCwxMjcsNzMsMjE3LDE5LDIxMyw1NCwxMzksNTcsMjQsMjI2LDgzLDE3NiwyNDcsMjIsNDksMTIwLDE4Nyw4MywxMSwyMTgsMTA4LDMxLDM0LDExOCwxMDIsMjIzLDE1NywyMzQsMjMsMTQ1LDIzLDYsMjUsNzEsMTQsNTcsMTg0LDE5NCwyMjQsNCw3MSwxODEsNzYsODcsMTU1LDM0LDI0Myw0MCw3Niw1MCwxMTIsMTEsMjMwLDI1NSwyNDgsMzcsNjQsNjUsMjksMjM1LDQ4LDU3LDE4NSwyMTEsMjEsMCwxMTUsMTUxLDEwMSw2NSwxMTgsMjEzLDExLDk1LDM0LDE2MywyMzIsMTg0LDc1LDI0MiwxOTgsMTAzLDE0OSw1OCwxMjIsNjUsMzQsNzgsNTYsMjI2LDQwLDQ1LDE0NiwxMjAsMTM4LDE2NSw5OSwyNTIsMzgsMTQsNjAsNjQsMCwyNTAsOTksNDYsMjQ5LDY2LDc3LDE5OSw3NSwyMTYsOCwxNDksOTQsMTExLDE1MCw1NSwxODMsMTUzLDkxLDE5MiwyMDAsMzMsNzIsMTAzLDM0LDIzMSw4OSwyNDcsODgsMTk1LDE1NSwxMDYsOTgsMjAsMTc1LDIwMCwxNjYsMTgwLDE1MiwxMjIsMTg3LDI3LDI5LDE5LDI5LDI0NCwxMDIsMTk3LDU0LDg5LDk0LDIzNiwzNSw3OSwyMzgsMzEsMTc4LDIxMCwyNDAsMTc5LDIwMyw1NywzNCw2MSw2Miw4NSw2OCw2OSwxNTUsMTcwLDEzMSw5NiwxNDcsMTcwLDEwNCwxMTMsMTUsMTUyLDAsOTUsMTYxLDE0MSw2MCwyMDUsNiw5MSwyNTUsMjQwLDcwLDYwLDc4LDIwLDI1NSwxMTAsNjksMjI2LDI0LDY0LDIzLDU5LDIwOCwxODMsMTA5LDUwLDIwMywxOTAsMjI4LDIyMCwyMzQsMTkwLDIwOCwxNDMsMjMsMjMzLDEwNiwxNTcsNDksMTY2LDE4LDQ3LDEyMCw5MSwxNDAsMCwyMjYsOTcsMTQyLDIwOSwyLDI1MSwyMzEsMTc1LDk1LDE1MCwyMjQsMjA1LDE1MCwxNDIsMjEzLDU5LDM4LDIzMCwyMDUsMTYxLDE5LDIwLDU0LDE3Nyw3NywxMzIsMTMxLDExLDE3NCwyMSwyMDQsOTIsMjI1LDI0MSwxMDMsMjAzLDE5MiwzMywxNTAsMjA4LDExMCw0LDIxNyw4MiwxODEsNTcsMTgwLDExMCwyMzAsMTk3LDIzNiw4NywxMCwxMjYsMTc3LDIwNywxMTUsMTgwLDE0Miw1MSwyMzYsMTI4LDExLDIwMCwyOCw4NSwxMCw5NSw5MSwyOCwxNSw0NywxMzgsNTQsMjksMjI2LDE3OCwyMzksOCwxNjksMzAsMTQwLDEzMSwxMDUsMTI4LDIsNTIsNTIsMzksMjAzLDEzNiwxOTQsNDUsMjEzLDIyLDI0MiwyMjUsMTQxLDEzNSwyNTIsOTQsMjA4LDE3Myw4Niw4Myw3NCwxNTAsODYsMjQwLDIzMSwyMywxODMsMTczLDk0LDE1MywxNzQsMTk5LDE0OCwxMzYsMTY3LDk3LDMzLDIwMCwyMzEsMjMyLDIyMywzMSw0MSwyNDcsMTQ2LDIyMiw0NCwyMzksNTcsOTksMTA5LDE2NywxNTksMTE1LDE2NCw0NCw0MiwxMDIsMTQ3LDIwNCwyNTEsODQsOTYsODgsMTkyLDIyNiwxOTcsMjUsMjI5LDE0NSwxOTMsNTQsMTg2LDEwNSwxMzgsMTI5LDIwMCwyNDEsMTAwLDE1OSw2OSw2LDYsMTczLDE5OCwyNTAsNjAsOTIsMjIzLDE2NCw2LDE0Myw1MSwyNTQsMTMyLDEyOCwxODksOTksMTE5LDU4LDEzMCwzOSw4NiwxMTYsMjMyLDMsMCwwLDAsMCwwLDAsNDgsMjMsMTksMTkwLDczLDQwLDEyNiw4NSw2Nyw5MywxOTksMTQ3LDg4LDIyMiw3NiwxMiwxMzcsNDYsNTYsNiwxNzgsNDEsNTMsMTUzLDExNSwxOTIsMTk5LDEyLDU3LDE4NywyNTUsNywxNDAsNjAsMTgzLDE3MiwyMTQsODksMjgsMTgwLDExNCwxNzksNTIsMTUxLDMwLDE3NCw5NCwyMSw0MywxMzAsMTAyLDIyMiwyMDgsMTYyLDIwMiw5MSwyMTksNzEsMjExLDEzNiwxMTMsMjM2LDI1MSwyMTcsNTIsMTI4LDExNiwxNTYsMTUyLDIxLDIwNiwxMTAsMTM1LDIyLDEzOCwxMTEsNTIsMTYsMTczLDExOSwxNzQsODMsNjEsMjM1LDE3NywyMTgsMTg2LDEyMiwxNzcsMjE3LDg1LDEyMSwyMTIsMTg5LDE3MywxNTUsMTEyLDI0MiwxNTksMjA0LDE5OCwyMTAsMTM2LDE5MCwxMSwyMzMsMjIyLDMyLDE5MiwzLDEzMCw1MywxNDEsMjQ5LDE2NywxNDMsMTkxLDE5Miw2NywyMDEsMTExLDE0LDIxNCwyMDIsMTk0LDEzOCwyMSwyMjAsMyw0MCw0Nyw3NiwxMTYsMTc1LDE4NCwxOTUsMTk3LDExMSwxMTgsMTA0LDExMiwyMzYsMjUyLDAsMzksMTU2LDM3LDE2OSwxMywyNDcsMTAzLDE1LDIxMiwyMDksMjQ4LDkyLDQ1LDE1NCwxOTYsMjE3LDM5LDExOCwyNDUsMTIyLDE5OSwyMzksMTYzLDEyNCwyMzcsNTQsMjUwLDQyLDExOCw0NywyNDMsMTA3LDE3MCwxNzksNTUsNiwyMTgsMTMzLDExOCwxMDYsMjM2LDE4MCwxNTEsMTY0LDEwLDIxMywyMTksMTQ2LDg5LDExNCw4MCwxNjQsMTE5LDIxNywxMTYsNDUsMjQ3LDI5LDE5OCwxMTYsMTc4LDEzNCw3OCw4LDIwNyw2NywxNDEsNTAsMjMsMTkyLDExNSw0MywxMDMsMTk4LDE4OSwxNjIsNDQsMTczLDEwMSwxMzgsMTYwLDE5OSwxNzksNjYsMTIyLDIyOSwxMDksMTEwLDIyOCwxNDQsNTYsMTY2LDU0LDExNywyMjUsOSwxMTYsMTgwLDE1MiwxNDgsNTYsODIsNjAsOTEsMjM1LDEwMiw1NCwyNTMsMTM3LDE5MywzMSw4Niw2NCwxMTAsNyw5OSwxOCwxODEsOCw5MCwyMyw4NSwyMzcsMTExLDg3LDYyLDEwOSw3MSwyMDEsNDMsMjM4LDk0LDE3NSwxOTYsMjAxLDg2LDI1NCwyOCwxNjksMjI2LDE4OCwxODUsNjUsMSwxNDAsNTgsMTExLDE3MSwyMiw0NywxNTcsMTE4LDExNiwyMTgsMjE2LDE2MCwyMCwyNDAsMTM3LDE5NSwyNDksNiwyMTEsMzQsNCwyNDksMTUxLDIxNiw5NSw1NSwxNDAsMTkyLDgxLDM2LDQ3LDE2NSwyNDcsNTcsMjEzLDQxLDE0NSwxODQsMTI5LDUzLDEwOSw1NiwyNTQsMTYwLDc1LDQsNjUsMzYsMTQ0LDE1LDk5LDEyMiwyNTQsNjgsOTksMTUzLDUzLDIxMiwyMzIsNjYsMjAwLDE0MSwxNDUsMzQsMTE4LDIwNCwxODEsNzIsMzQsMTQzLDIyOSwxMCwxNjQsMTUwLDIyMiw0Niw2LDIxNywyMjYsNCwxMjYsMjA3LDE5MywxMzgsMjQ1LDEyNiw2MSwxNCw3NSw3MywxNTAsMTM0LDE1OCw1MSwzNywyMTcsMTMsMTA3LDE0NCwyMjcsMjEsMTk0LDEyMiwyNDQsMjQzLDE2NSwxNzAsOSwyNDUsODcsMTQ1LDE3MCw2MiwxMzEsMTYxLDkwLDEzMCwxMTcsMjExLDIyNywyNDgsODgsNjksMTgyLDEwOCw3Niw4OSwxMDksMTM3LDE1MywxNCwyNTEsNDAsMTksMTE2LDEwNCwyNiwxMjYsMTU1LDk2LDE0MCwxMzgsMTExLDk2LDE0MiwxNjQsMTcyLDEyMCw5MiwxMjIsOTMsMjEwLDEyNiwzOCwzOSwyNTQsMTY0LDEwMCwyMjQsMjEyLDE4NCw0Myw5LDIwLDQ0LDczLDI0NSwyNTQsMTQzLDIwMiwyNDYsMjEyLDIwNCw5MywxNTQsODIsMjIzLDc1LDE4LDE4MCwxMDQsMTQzLDI1MSwxNTksMjQyLDExNiw0NSw2MiwxNzAsMzgsMTI5LDIxLDE3NiwxNzIsMjcsMTAyLDE4LDIxNSw1MSwxODUsNzgsMTIzLDE4NSw5MSwxNjgsMTIxLDExMCwxMTAsMTA3LDYzLDEyMiw3OCwxNzcsNDYsMTY4LDM0LDIzNCwwLDExNCw2MiwyMDQsMTU3LDYsMjEzLDIyOSwxMCw3OSwxOTcsMTYwLDE1MywyMzQsOTIsMjA1LDE3NSwxNjQsNDUsMzcsNjksNDgsNDAsMTI4LDEwOCw5Niw3NCwxODQsMTk5LDIxOCwxNzYsMjEsMzYsODksNTgsMTMwLDIxNSw4MCwzMCwyMDQsMjQ3LDEyMCw5LDIwLDE0NSwxNSwzNiwxMzcsMTMsMjA3LDE0MSw0NCwxMzEsMTI2LDUwLDM4LDI0NywyNTUsNDMsMzYsMTYxLDE5OSwxNzMsMzUsMjQsMTU1LDI0MSw2NCw4MCw4MSwxNjYsMTUsNDgsMzksMjQ0LDIyNCwzNiwxNzYsMzgsMTksMTk2LDI0NCw2LDE0Miw3NSw2MSwxNTYsMzAsMTY4LDQ2LDI3LDIzMSwxOTgsMTE3LDI0MiwyMzIsMTk1LDM3LDkzLDM3LDI1NSwxMjIsMTk4LDExNSwyMjQsMzgsNzYsMjE3LDk0LDEsMTE0LDE1LDEzMSw4Niw4NSwxNzEsMTc3LDI0MCwxMjUsMTY4LDAsMjEwLDEzNywxNzksMTg3LDQwLDIyMiw2NCwxMDUsNzAsMTMsMjYsMjAwLDIyMiwyMjMsMzYsNDEsMTkxLDc0LDEwOSwxMjMsNjMsODEsNzIsMjA2LDEwNSw3MSwxNjksMTM3LDE5NCwxMjgsMjMxLDIzNSw4MSwzMiwxNDUsNzIsNTMsMjA5LDkyLDE5NSwxMzMsMTM3LDMwLDgsMjYsMjI3LDUyLDE4OSwxOCwxMzYsMjQ0LDczLDYzLDE3NiwyMTAsNTcsMTA0LDYxLDI0MCwxNTksMzksMTU2LDYsMjU0LDEyOSwyMTYsMjMsNCwxNCw1NCw3MywxOTEsMjI0LDEyNCwyNDcsNjUsOSwyMDYsMTk5LDUwLDIwNiwyMTEsMzgsNSw2LDIzNiwyNTQsMjE1LDE5NywxMjQsMjIsMTkzLDYxLDIwMSwxMzUsMTg4LDIyOCwxODcsMTEyLDQ1LDIzNywxMTUsODYsMjE5LDE2Myw0NCw3OCw2MiwyNDMsMywwXQ=="
    },
    "id": 1
}
```

### zapavm.nodeBlockCounts

Get information about which nodes have produced how many blocks

#### Result

```
{
  `"NodeBlockCounts"    Dictionary(string->integer)`  A dictionary indicating how many blocks each node has produced. Keys are node IDs and values are block counts.
}
```

#### Example

##### Request

```
curl --location --request POST 'http://$HOST:$PORT/ext/bc/$BLOCKCHAIN' \--header 'Content-Type: application/json' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "zapavm.NodeBlockCounts",
    "params":{},
    "id": 1
}
'
```


##### Response

```
{
    "jsonrpc": "2.0",
    "result": {
        "NodeBlockCounts": {
            "46BWRmkG6audrPEGC2qMqd1yV7fKtvyH3": 101,
            "FfYHcLqA1DCabUos5uQUJmr98cG3aWBsQ": 3,
            "NLaFpb6wy6bJHfit5f4pYS3r8uEGnMULV": 11
        }
    },
    "id": 1
}
```
