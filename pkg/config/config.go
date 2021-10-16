package config

import (
	"encoding/json"
	"io/ioutil"
)

const (
	Version = "0.0.1"
	HTTPFLV = "GOSM/flv_0.0.1"
	HLS     = "GOSM/hls_0.0.1"
)

type Config struct {
	RTMP    RTMPCfg    `json:"rtmp"`
	HTTPFLV HTTPFlvCfg `json:"http_flv"`
	HLS     HLSCfg     `json:"hls"`
	RTP     RTP        `json:"rtp"`

	LogLevel     uint8 `json:"log_level"`
	MachineID    int64 `json:"machine_id"`
	DataCenterID int64 `json:"datacenter_id"`
}

type RTMPCfg struct {
	Port          string `json:"port"`
	GopSize       uint8  `json:"gop_size"`
	AVReadTimeout int64  `json:"read_timeout"`
}

type HTTPFlvCfg struct {
	Enable bool   `json:"enable"`
	Port   string `json:"port"`
}

type HLSCfg struct {
	Enable     bool   `json:"enable"`
	Port       string `json:"port"`
	TsPath     string `json:"ts_path"`
	TsPrefix   string `json:"ts_prefix"`
	TsDuration int    `json:"ts_duration"`
	TsWindow   int    `json:"ts_window"`
}

type RTP struct {
	Enable      bool     `json:"enable"`
	Remote      string   `json:"remote"`
	Ports       []string `json:"ports"`
	ReadTimeout int64    `json:"read_timeout"`
}

var Global = &Config{}

func init() {
	data, err := ioutil.ReadFile("../configs/config.json")
	if err != nil {
		panic(err)
	}

	if err := json.Unmarshal(data, Global); err != nil {
		panic(err)
	}
}
