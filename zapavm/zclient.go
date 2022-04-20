package zapavm

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	log "github.com/inconshreveable/log15"

	nativejson "encoding/json"
)

const (
	zcashHost  = "127.0.0.1"
	zcashPort = 8232
	zcashUser  = "test"
	zcashPw    = "pw"
)

type ZCashResponse struct {
	Result nativejson.RawMessage `json:"result`
	Error  string                `json:"error"`
	ID     string                `json:"id"`
}

type ZCashRequest struct {
	Params []interface{} `json:"params"`
	Method string        `json:"method"`
	ID     string        `json:"id"`
}

type ZCashRequest2 struct {
	Params nativejson.RawMessage `json:"params"`
	Method string                `json:"method"`
	ID     string                `json:"id"`
}

type ZcashClient struct {
	Host     string
	Port     int
	User     string
	Password string
	Mock     bool
}

func (zc *ZcashClient) GetHost() string {
	if zc.Host == "" {
		return zcashHost
	}
	return zc.Host
}

func (zc *ZcashClient) GetPort() int {
	if zc.Port == 0 {
		return zcashPort
	}
	return zc.Port
}

func (zc *ZcashClient) GetUser() string {
	if zc.User == "" {
		return zcashUser
	}
	return zc.User
}

func (zc *ZcashClient) GetPassword() string {
	if zc.Password == "" {
		return zcashPw
	}
	return zc.Password
}

func (zc *ZcashClient) GetCompleteHost() string {
	return "http://" + zc.GetUser() + ":" + zc.GetPassword() + "@" + zc.GetHost() + ":" + strconv.Itoa(zc.GetPort())
}
 
func (zc *ZcashClient) ZcashSendMany(from string, to string, amount float32) ZCashResponse {
	log.Info("Calling Zcash Method: z_sendmany", "from", from, "to", to, "amount", amount)
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

func (zc *ZcashClient) GetZcashResponse(b []byte) ZCashResponse {
	dataz := string(b)

	completeHost := zc.GetCompleteHost()
	serializedData := strings.NewReader(dataz)
	log.Info("Connecting to zcash", "Compelte Host", completeHost, "data", serializedData)
	resp, err := http.Post(completeHost, "application/json", serializedData)
	if err != nil {
		log.Error("Post: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("ReadAll: %v", err)
	}
	zresp := ZCashResponse{}
	nativejson.Unmarshal([]byte(body), &zresp)
	// err = nativejson.Unmarshal(body, &result)
	if err != nil {
		log.Error("Unmarshal: %v", err)
	}
	return zresp
}

func (zc *ZcashClient) CallZcashJson(method string, params []interface{}) ZCashResponse {
	completeHost := zc.GetCompleteHost()
	log.Info("Calling Zcash Json", "Method", method, "params", params, "complete host", completeHost)

	req := &ZCashRequest{Params: params, Method: method, ID: "fromgo"}
	b, err := nativejson.Marshal(req)
	if err != nil {
		fmt.Println(err)
	}
	return zc.GetZcashResponse(b)
}

func (zc *ZcashClient) GetBlockCount() int {
	if MockZcash {
		return 20
	}
	blkcnt :=  zc.CallZcash("getblockcount", nil)
	r, _ := strconv.Atoi(string(blkcnt.Result))
	return r
}

func (zc *ZcashClient) GetZBlock(height int) nativejson.RawMessage {
	if MockZcash {
		log.Info("Calling mock get z block", "block num", height)
		plan, _ := ioutil.ReadFile("/Users/rkass/repos/zapa/zapavm/zapavm/mocks/block" + strconv.Itoa(height + 1) + ".json")
		return plan
	}
	resp := zc.CallZcashJson("getserializedblock", []interface{}{strconv.Itoa(height)})
	return resp.Result
}

func (zc *ZcashClient) BlockGenerator() chan nativejson.RawMessage {
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


func (zc *ZcashClient) CallZcash(method string, zresult nativejson.RawMessage) ZCashResponse {
	if MockZcash && method == "validateBlock" {
		log.Info("Calling Mock zcash validate block", "params", zresult)
		return ZCashResponse{
			ID: "mockcall",
		}
	}
	log.Info("Calling Zcash", "Method", method, "params", zresult)
	var req *ZCashRequest2
	if zresult != nil {
		var x []uint8 = []uint8{}
		for _, i := range zresult {
			x = append(x, i)
		}
		req = &ZCashRequest2{Params: x, Method: method, ID: "fromgo"}
	} else {
		req = &ZCashRequest2{Params: nil, Method: method, ID: "fromgo"}
	}
	// b, err := req.MarshalJSON()
	b, err := nativejson.Marshal(req)
	if err != nil {
		fmt.Println(err)
	}
	return zc.GetZcashResponse(b)
}
