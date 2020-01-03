package appconfig

import (
	"github.com/BurntSushi/toml"
)

type InsaneConfigs struct {
	Http    HttpConfig `toml:"http"`
	Worker  Worker     `toml:"worker"`
	Log     Log        `toml:"log"`
	Cluster Cluster    `toml:"cluster"`
}

type HttpConfig struct {
	Bind                string `toml:"bind"`
	HttpOrigin          string `toml:"HttpOrigin"`      //= "Access-Control-Allow-Origin"
	HttpHeaders         string `toml:"HttpHeaders"`     //= "Access-Control-Allow-Headers"
	HttpContentType     string `toml:"HttpContentType"` //= "application/json"
	MaxIdleConnsPerHost int    `toml:"MaxIdleConnsPerHost"`
}

type Cluster struct {
	MasterUrl string `toml:"masterUrl"`
}

type Worker struct {
	TaskLife uint64 `toml:"taskLife"`
}

type Log struct {
	Location string `toml:"location"`
}

var cnf InsaneConfigs

func InitConfig(path string) error {
	if _, err := toml.DecodeFile(path, &cnf); err != nil {
		return err
	}
	return nil
}

func GetConfig() *InsaneConfigs {
	return &cnf
}
