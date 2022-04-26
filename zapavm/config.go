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
}

func NewChainConfig(conf []byte) ChainConfig {
	cconf := ChainConfig{
		Enabled: true,
		MockZcash: false,
	}
	as := os.Getenv("AVASIM")
	if as != "" {
		log.Info("Running as part of local ava-sim environment")
		cconf.AvaSim = true
	}
	jsonerr := nativejson.Unmarshal(conf, &cconf)
	if jsonerr != nil {
		log.Warn("error initializing config, returning default config")
	}
	return cconf
}

func (c *ChainConfig) ZcashClient(nodeID string) (zclient.ZcashClient, error) {
	if c.MockZcash {
		log.Info("initializing mock zcash client")
		return zclient.NewDefaultMock(), nil
	}
	if c.AvaSim {
		log.Info("initializing local node config by examining ~/node-ids/ directory")
		i := 0
		h, _ := os.LookupEnv("HOME")
		for i < 6 {
			fname := h + "/node-ids/" + strconv.Itoa(i)
			nid, _ := ioutil.ReadFile(fname)
			snid := strings.ReplaceAll(string(nid), "NodeID-", "")
			log.Info("comparing", "file name", fname, "fvalue", snid, "nid", nodeID)
			if snid == nodeID {
				log.Info("Initializing zcash client as node num", "num", i)
				return &zclient.ZcashHTTPClient {
					Host: "127.0.0.1",
					Port: 8232 + i + 1,
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
