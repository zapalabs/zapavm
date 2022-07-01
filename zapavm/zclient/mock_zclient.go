package zclient

import (
	nativejson "encoding/json"
	"io/ioutil"
	"os"
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
	log.Warn("ZCMockClient.SetHost is a no-op", "host", host)
}

func (zc *ZCashMockClient) SetPort(port int) {
	log.Warn("ZCMockClient.SetPort is a no-op", "port", port)
}

func (zc *ZCashMockClient) SendMany(from string, to string, amount float32) ZCashResponse {
	errString := "Calling send many with mock client. This is a no-op."
	log.Warn(errString, "from", from, "to", to, "amount", amount)

	return ZCashResponse{Error: &ZcashError{Message: errString, Code: ZcashClientErrorCode}}

}

func (zc *ZCashMockClient) GetBlockCount() (int, error) {
	log.Info("ZCMockClient.GetBlockCount")
	return zc.InitialBlocks, nil
}

func (zc *ZCashMockClient) GetZBlock(height int) ZcashBlockResult {
	log.Info("ZCMockClient.GetZBlock", "height", height)
	h, _ := os.LookupEnv("HOME")
	fname := h + "repos/zapa/zapavm/zapavm/mocks/block" + strconv.Itoa(height + 1) + ".json"
	plan, _ := ioutil.ReadFile(fname)
	return ZcashBlockResult{
		Block: plan,
		Timestamp: int64(height),
	}
}

func (zc *ZCashMockClient) ValidateBlocks(zblk nativejson.RawMessage) error {
	log.Info("ZCMockClient.ValidateBlocks. Naively returning nil indicating a valid block")
	return nil
}

func (zc *ZCashMockClient) SubmitBlock(zblk nativejson.RawMessage) error {
	log.Info("ZCMockClient.Submit. Naively returning nil indicating a success")
	return nil
}

func (zc *ZCashMockClient) SuggestBlock() ZcashBlockResult {
	log.Warn("ZCMockClient.SuggestBlock. Returning empty response")
	return ZcashBlockResult{}
}

func (zc *ZCashMockClient) CallZcash(method string, zresult nativejson.RawMessage) ZCashResponse {
	errString := "ZCMockClient.CallZcash with mock client. This is a no-op."
	log.Warn(errString)

	return ZCashResponse{Error: &ZcashError{Message: errString, Code: ZcashClientErrorCode}}
}

func (zc *ZCashMockClient) CallZcashJson(method string, params []interface{}) ZCashResponse {
	errString := "Calling CallZcashJson with mock client. This is a no-op."
	log.Warn(errString)

	return ZCashResponse{Error: &ZcashError{Message: errString, Code: ZcashClientErrorCode}}

}