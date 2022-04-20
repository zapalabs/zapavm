package zclient

import (
	nativejson "encoding/json"
)

type ZCashResponse struct {
	Result nativejson.RawMessage `json:"result"`
	Error  string                `json:"error"`
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

type ZcashClient interface {
	SetHost(host string)
	SetPort(port int)
	SendMany(from string, to string, amount float32) ZCashResponse
	GetBlockCount() int
	GetZBlock(height int) nativejson.RawMessage
	ValidateBlock(zblk nativejson.RawMessage) error
	SubmitBlock(zblk nativejson.RawMessage) error
	SuggestBlock() ZCashResponse
	CallZcash(method string, zresult nativejson.RawMessage) ZCashResponse
	CallZcashJson(method string, params []interface{}) ZCashResponse
}

func BlockGenerator(zc ZcashClient) chan nativejson.RawMessage {
	c := make(chan nativejson.RawMessage)
	go func() {
		numBlks := zc.GetBlockCount()
		blkcnt := 0
		for blkcnt <= numBlks {
			c <- zc.GetZBlock(blkcnt)
			blkcnt++
		}
		close(c)
	}()
	return c
}
