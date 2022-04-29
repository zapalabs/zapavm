package zapavm

import (
	nativejson "encoding/json"
	"fmt"

	log "github.com/inconshreveable/log15"

	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/zapalabs/zapavm/zapavm/zclient"
)

type ChainConfig struct {
	Enabled  bool `json:"enabled"`
	MockZcash bool `json:"mockZcash"`
	AvaSim    bool `json:"avasim"`
    ZcashHost  string  `json:"zcashHost"`
    ZcashPort int `json:"zcashPort"`
	ZcashUser string `json:"zcashUser"`
	ZcashPassword string `json:"zcashPassword"`
	ClearDatabase bool `json:"clearDatabase"`
	LogLevel string `json:"logLevel"`
}

func NewChainConfig(conf []byte) ChainConfig {
	cconf := ChainConfig{
		Enabled: true,
		MockZcash: false,
		LogLevel: log.LvlInfo.String(),
	}
	as := os.Getenv("AVASIM")
	if as != "" {
		log.Info("Running as part of local ava-sim environment")
		cconf.AvaSim = true
	}
	jsonerr := nativejson.Unmarshal(conf, &cconf)
	if jsonerr != nil {
		log.Warn("Error initializing config, returning default config")
		log.Debug("Config initialization error", "error", jsonerr)
	}
	return cconf
}

func (c *ChainConfig) ZcashClient(nodeID string) (zclient.ZcashClient, error) {
	if c.MockZcash {
		log.Info("Initializing mock zcash client")
		return zclient.NewDefaultMock(), nil
	}
	if c.AvaSim {
		log.Info("Initializing local node config by examining ~/node-ids/ directory")
		i := 0
		h, _ := os.LookupEnv("HOME")
		for i < 6 {
			// examine each file in ~/node-ids/ dir. is the file name
			// equal to our node id? if so, we assume that node number.
			// once we know our node number our zcash port becomes known
			fname := h + "/node-ids/" + strconv.Itoa(i)
			nid, _ := ioutil.ReadFile(fname)
			snid := strings.ReplaceAll(string(nid), "NodeID-", "")
			if snid == nodeID {
				port := 8233 + i + 1
				log.Info("Initializing zcash client", "node number", i, "zcash port", port)
				return &zclient.ZcashHTTPClient {
					Host: "127.0.0.1",
					Port: port,
					User: "test",
					Password: "pw",
				}, nil
			}
			i += 1
		}
		return &zclient.ZcashHTTPClient{}, fmt.Errorf("Unable to initialize node config from reading ~/node-ids directory")
	}
	return &zclient.ZcashHTTPClient{
		Host: c.ZcashHost,
		Port: c.ZcashPort,
		User: c.ZcashUser,
		Password: c.ZcashPassword,
	}, nil
}
