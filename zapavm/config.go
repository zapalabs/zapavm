package zapavm

type VMConfig struct {
    ZcashHost  string  `json:"zcashHost"`
    ZcashPort int `json:"zcashPort"`
	ZcashUser string `json:"zcashUser"`
	ZcashPassword string `json:"zcashPassword"`
}