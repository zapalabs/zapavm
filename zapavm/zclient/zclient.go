package zclient

import (
	nativejson "encoding/json"

	log "github.com/inconshreveable/log15"
)

type ZCashResponse struct {
	Result nativejson.RawMessage `json:"result"`
	Error  error                 `json:"error"`
	ID     string                `json:"id"`
}

type ZCashRequest struct {
	Params []interface{} `json:"params"`
	Method string        `json:"method"`
	ID     string        `json:"id"`
}

type ZCashRequestJson struct {
	Params nativejson.RawMessage `json:"params"`
	Method string                `json:"method"`
	ID     string                `json:"id"`
}

type ZcashBlockResult struct {
	Block nativejson.RawMessage `json:"block"`
	Timestamp int64             `json:"timestamp"`
	Error error
}

type ZcashClient interface {
	SetHost(host string)
	SetPort(port int)
	SendMany(from string, to string, amount float32) ZCashResponse
	GetBlockCount() (int, error)
	GetZBlock(height int) ZcashBlockResult
	ValidateBlock(zblk nativejson.RawMessage) error
	SubmitBlock(zblk nativejson.RawMessage) error
	SuggestBlock() ZCashResponse
	CallZcash(method string, zresult nativejson.RawMessage) ZCashResponse
	CallZcashJson(method string, params []interface{}) ZCashResponse
}

func BlockGenerator(zc ZcashClient) chan ZcashBlockResult {
	c := make(chan ZcashBlockResult)
	var e error
	go func() {
		numBlks, err := zc.GetBlockCount()
		if err != nil {
			e = err
			return
		}
		blkcnt := 0
		for blkcnt <= numBlks {
			c <- zc.GetZBlock(blkcnt)
			blkcnt++
		}
		close(c)
	}()
	log.Error("error generating blocks", "error", e)
	return c
}
