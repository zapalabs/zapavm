package zclient

import (
	nativejson "encoding/json"
	"io/ioutil"
	"strconv"

	log "github.com/inconshreveable/log15"
)

const (
	DefaultInitialBlocks = 15
)

type ZCashMockClient struct {
	InitialBlocks int
}

func NewDefaultMock() *ZCashMockClient {
	return &ZCashMockClient{
		InitialBlocks: DefaultInitialBlocks,
	}
}

func (zc *ZCashMockClient) SetHost(host string) {
	log.Warn("Calling set host on mock client. This is a no-op.", "host", host)
}

func (zc *ZCashMockClient) SetPort(port int) {
	log.Warn("Calling set port on mock client. This is a no-op.", "port", port)
}

func (zc *ZCashMockClient) SendMany(from string, to string, amount float32) ZCashResponse {
	errString := "Calling send many with mock client. This is a no-op."
	log.Warn(errString, "from", from, "to", to, "amount", amount)

	return ZCashResponse{
		Error: errString,
	}
}

func (zc *ZCashMockClient) GetBlockCount() int {
	log.Info("Calling ZcashMockClient GetBlockCount")
	return zc.InitialBlocks
}

func (zc *ZCashMockClient) GetZBlock(height int) nativejson.RawMessage {
	log.Info("Calling ZcashMockClient GetZBlock", "height", height)
	plan, _ := ioutil.ReadFile("/Users/rkass/repos/zapa/zapavm/zapavm/mocks/block" + strconv.Itoa(height + 1) + ".json")
	return plan
}

func (zc *ZCashMockClient) ValidateBlock(zblk nativejson.RawMessage) error {
	log.Info("Calling ZcashMockClient ValidateBlock. Naively returning nil indicating a valid block")
	return nil
}

func (zc *ZCashMockClient) SubmitBlock(zblk nativejson.RawMessage) error {
	log.Info("Calling ZcashMockClient Submit. Naively returning nil indicating a success")
	return nil
}

func (zc *ZCashMockClient) SuggestBlock() ZCashResponse {
	log.Warn("Calling ZCashMockClient suggest block. Returning empty response")
	return ZCashResponse{}
}

func (zc *ZCashMockClient) CallZcash(method string, zresult nativejson.RawMessage) ZCashResponse {
	errString := "Calling CallZcash with mock client. This is a no-op."
	log.Warn(errString)

	return ZCashResponse{
		Error: errString,
	}
}

func (zc *ZCashMockClient) CallZcashJson(method string, params []interface{}) ZCashResponse {
	errString := "Calling CallZcashJson with mock client. This is a no-op."
	log.Warn(errString)

	return ZCashResponse{
		Error: errString,
	}
}