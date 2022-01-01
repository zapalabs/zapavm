package zapavm

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/inconshreveable/log15"

	nativejson "encoding/json"
)

const (
	zcashHost  = "127.0.0.1"
	zcash0Port = "8232"
	zcash1Port = "8233"
	zcash2Port = "8234"
	zcash3Port = "8235"
	zcash4Port = "8236"
	zcashUser  = "test"
	zcashPw    = "pw"
)

func zcashCompleteHost(zcashInstance int) string {
	if zcashInstance == 0 {
		return "http://" + zcashUser + ":" + zcashPw + "@" + zcashHost + ":" + zcash0Port
	} else if zcashInstance == 1 {
		return "http://" + zcashUser + ":" + zcashPw + "@" + zcashHost + ":" + zcash1Port
	} else if zcashInstance == 2 {
		return "http://" + zcashUser + ":" + zcashPw + "@" + zcashHost + ":" + zcash2Port
	} else if zcashInstance == 3 {
		return "http://" + zcashUser + ":" + zcashPw + "@" + zcashHost + ":" + zcash3Port
	} else if zcashInstance == 4 {
		return "http://" + zcashUser + ":" + zcashPw + "@" + zcashHost + ":" + zcash4Port
	}
	return "unknown"
}

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

func ZcashSendMany(from string, to string, amount float32, zcashInstance int) ZCashResponse {
	log.Info("Calling Zcash Method: z_sendmany", "from", from, "to", to, "amount", amount, "instance num", zcashInstance)
	var params []interface{}
	params = append(params, from)
	var destination []interface{}
	var dest map[string]interface{} = make(map[string]interface{})
	dest["address"] = to
	dest["amount"] = amount
	destination = append(destination, dest)
	params = append(params, destination)
	return CallZcashJson("z_sendmany", params, zcashInstance)
}

func GetZcashResponse(b []byte, zcashInstance int) ZCashResponse {
	dataz := string(b)

	completeHost := zcashCompleteHost(zcashInstance)
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

func CallZcashJson(method string, params []interface{}, zcashInstance int) ZCashResponse {
	log.Info("Calling Zcash Json", "Method", method, "params", params, "nodenum", zcashInstance)

	req := &ZCashRequest{Params: params, Method: method, ID: "fromgo"}
	b, err := nativejson.Marshal(req)
	if err != nil {
		fmt.Println(err)
	}
	return GetZcashResponse(b, zcashInstance)
}

func CallZcash(method string, zresult nativejson.RawMessage, zcashInstance int) ZCashResponse {
	log.Info("Calling Zcash", "Method", method, "params", zresult, "nodenum", zcashInstance)
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
	return GetZcashResponse(b, zcashInstance)
}
