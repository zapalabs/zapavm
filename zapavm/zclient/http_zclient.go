package zclient

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	log "github.com/inconshreveable/log15"

	nativejson "encoding/json"
)

type ZcashHTTPClient struct {
	Host     string
	Port     int
	User     string
	Password string
}

const (
	ZcashHost  = "127.0.0.1"
	ZcashPort = 8232
	ZcashUser  = "test"
	ZcashPw    = "pw"
)

func (zc *ZcashHTTPClient) GetHost() string {
	if zc.Host == "" {
		return ZcashHost
	}
	return zc.Host
}

func (zc *ZcashHTTPClient) GetPort() int {
	if zc.Port == 0 {
		return ZcashPort
	}
	return zc.Port
}

func (zc *ZcashHTTPClient) GetUser() string {
	if zc.User == "" {
		return ZcashUser
	}
	return zc.User
}

func (zc *ZcashHTTPClient) GetPassword() string {
	if zc.Password == "" {
		return ZcashPw
	}
	return zc.Password
}

func (zc *ZcashHTTPClient) SetHost(host string) {
	zc.Host = host
}

func (zc *ZcashHTTPClient) SetPort(port int)  {
	zc.Port = port
}

func (zc *ZcashHTTPClient) GetCompleteHost() string {
	return "http://" + zc.GetUser() + ":" + zc.GetPassword() + "@" + zc.GetHost() + ":" + strconv.Itoa(zc.GetPort())
}
 
func (zc *ZcashHTTPClient) SendMany(from string, to string, amount float32) ZCashResponse {
	log.Info("Calling ZcashHttpClient method: SendMany", "from", from, "to", to, "amount", amount)
	var params []interface{}
	params = append(params, from)
	var destination []interface{}
	var dest map[string]interface{} = make(map[string]interface{})
	dest["address"] = to
	dest["amount"] = amount
	destination = append(destination, dest)
	params = append(params, destination)
	return zc.CallZcashJson("z_sendmany", params)
}

func (zc *ZcashHTTPClient) GetBlockCount() (int, error) {
	blkcnt :=  zc.CallZcash("getblockcount", nil)
	r, e := strconv.Atoi(string(blkcnt.Result))
	if blkcnt.Error == nil {
		return r, e
	}
	return r, blkcnt.Error.Error()
}

func (zc *ZcashHTTPClient) GetZBlock(height int) ZcashBlockResult {
	resp := zc.CallZcashJson("getserializedblock", []interface{}{strconv.Itoa(height)})
	return blockResultFromResp(resp)
}

func (zc *ZcashHTTPClient) CallZcashJson(method string, params []interface{}) ZCashResponse {
	log.Info("ZcashHTTPClient.CallZcashJson", "Method", method, "Params", params, "Complete Host", zc.GetCompleteHost())

	req := &ZCashRequest{Params: params, Method: method, ID: "fromgo"}
	b, err := nativejson.Marshal(req)
	if err != nil {
		errstr := "Error marshalling request to json"
		log.Error(errstr, "error", err)
		return ZCashResponse{Error: &ZcashError{Message: errstr, Code: ZcashClientErrorCode}}
	}
	return zc.getZcashResponse(b)
}

func (zc *ZcashHTTPClient) ValidateBlock(zblk nativejson.RawMessage) error {
	r := zc.CallZcash("validateBlock", zblk)
	if r.Error != nil {
		log.Error("validate block call did not succeed", "error", r.Error)
		return r.Error.Error()
	}
	s := string(r.Result[:])
	if s != "null" {
		log.Error("validate block returned error", "s", s)
		return fmt.Errorf("error validating block")
	}
	return nil
}

func (zc *ZcashHTTPClient) SubmitBlock(zblk nativejson.RawMessage) error {
	resp := zc.CallZcash("submitblock", zblk)
	if resp.Error != nil {
		return resp.Error.Error()
	}
	return nil
}

func (zc *ZcashHTTPClient) SuggestBlock() ZcashBlockResult {
	resp := zc.CallZcash("suggest", nil)
	return blockResultFromResp(resp)
}

func (zc *ZcashHTTPClient) CallZcash(method string, zresult nativejson.RawMessage) ZCashResponse {
	log.Info("ZcashHTTPClient.CallZcash", "Method", method, "Complete Host", zc.GetCompleteHost())
	
	var req *ZCashRequestJson
	if zresult != nil {
		var x []uint8 = []uint8{}
		for _, i := range zresult {
			x = append(x, i)
		}
		req = &ZCashRequestJson{Params: x, Method: method, ID: "fromgo"}
	} else {
		req = &ZCashRequestJson{Params: nil, Method: method, ID: "fromgo"}
	}

	b, err := nativejson.Marshal(req)
	if err != nil {
		errorstr := fmt.Sprintf("Error marshalling request: %e", err)
		log.Error(errorstr)
		return ZCashResponse{Error: &ZcashError{Message: errorstr, Code: ZcashClientErrorCode}}
	}

	return zc.getZcashResponse(b)
}

func blockResultFromResp(resp ZCashResponse) ZcashBlockResult {
	log.Debug("ZcashHTTPClient.blockResultFromResp: begin")
	var arr []ZcashBlockResult
	zbr := ZcashBlockResult{}
	err := nativejson.Unmarshal(resp.Result, &arr)
	if err != nil {
		log.Error("Error unmarshalling block result", "error", err)
		zbr.Error = err
		return zbr
	} else if len(arr) != 1 {
		errstr := fmt.Errorf("Received unexpected length of response. expected 1. received %d", len(arr))
		log.Error("error: %s", errstr)
		zbr.Error = errstr
		return zbr
	}
	return arr[0]
}

func (zc *ZcashHTTPClient) getZcashResponse(b []byte) ZCashResponse {
	dataz := string(b)
	completeHost := zc.GetCompleteHost()
	serializedData := strings.NewReader(dataz)
	log.Debug("Connecting to zcash", "Compelte Host", completeHost, "data", serializedData)
	resp, err := http.Post(completeHost, "application/json", serializedData)
	
	if err != nil {
		errorstr := fmt.Sprintf("Error getting zcash response. Complete Host: %s ; error: %e", completeHost, err)
		log.Error(errorstr)
		return ZCashResponse{Error: &ZcashError{Message: errorstr, Code: ZcashClientErrorCode}}
	}
	
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errorstr := fmt.Sprintf("Error reading zcash response. Complete Host: %s ; error: %e", completeHost, err)
		log.Error(errorstr)
		return ZCashResponse{Error: &ZcashError{Message: errorstr, Code: ZcashClientErrorCode}}
	}
	
	zresp := ZCashResponse{}
	nativejson.Unmarshal([]byte(body), &zresp)
	if err != nil {
		errorstr := fmt.Sprintf("Error unmarshalling zcash response. Complete Host: %s ; error: %e", completeHost, err)
		log.Error(errorstr)
		return ZCashResponse{Error: &ZcashError{Message: errorstr, Code: ZcashClientErrorCode}}
	}

	log.Debug("ZcashHttpClient.getZcashResponse: returning", "ZcashResponse", zresp)

	return zresp
}
