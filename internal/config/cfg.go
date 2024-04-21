// Copyright (C) 2022 AlgoNode Org.
//
// reti-algodrop is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// reti-algodrop is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with reti-algodrop.  If not, see <https://www.gnu.org/licenses/>.

package config

import (
	"flag"
	"fmt"

	"github.com/algonode/reti-algodrop/internal/utils"
)

var cfgFile = flag.String("f", "config.jsonc", "config file")

type NodeConfig struct {
	Address   string `json:"address"`
	Token     string `json:"token"`
	RateLimit int    `json:"ratelimit"`
}

type KV map[string]string
type KB map[string]bool

type GraphQLConfig struct {
	Address string `json:"address"`
}

type DBConfig struct {
	Dsn string `json:"dsn"`
}

type AlgoDropConfig struct {
	Sleep  int    `json:"cache"`
	PKey   string `json:"sender_key"`
	AKey   string `json:"auth_key"`
	Reward int    `json:"reward"`
}

type NotifierConfig struct {
	Algod  *NodeConfig     `json:"algod-api"`
	GQL    *GraphQLConfig  `json:"gql-api"`
	DB     *DBConfig       `json:"db"`
	ADrop  *AlgoDropConfig `json:"algodrop"`
	PKeys  KV              `json:"pkeys"`
	WSnglt KB              `json:"singletons"`
}

var defaultConfig = NotifierConfig{}

// loadConfig loads the configuration from the specified file, merging into the default configuration.
func LoadConfig() (cfg NotifierConfig, err error) {
	flag.Parse()
	cfg = defaultConfig
	err = utils.LoadJSONCFromFile(*cfgFile, &cfg)

	if cfg.Algod == nil {
		return cfg, fmt.Errorf("[CFG] Missing algod config")
	}

	if cfg.GQL == nil {
		return cfg, fmt.Errorf("[CFG] Missing gql config")
	}

	if cfg.PKeys == nil {
		return cfg, fmt.Errorf("[CFG] Missing pkeys config")
	}

	if cfg.ADrop == nil {
		return cfg, fmt.Errorf("[CFG] Missing votescan config")
	}

	if cfg.WSnglt == nil || len(cfg.WSnglt) == 0 {
		return cfg, fmt.Errorf("[CFG] Singleton config missing")
	}

	return cfg, err
}
