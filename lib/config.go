//----------------------------------------------------------------------
// This file is part of 'Adresser'.
// Copyright (C) 2021 Bernd Fix >Y<
//
// 'Adresser' is free software: you can redistribute it and/or modify it
// under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// 'Addresser' is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
//
// SPDX-License-Identifier: AGPL3.0-or-later
//----------------------------------------------------------------------

package lib

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

// CoinConfig is a configuration for a supported coin (Bitcoin or Altcoin)
type CoinConfig struct {
	Name  string `json:"name"`  // coin symbol
	Descr string `json:"descr"` // coin description
	Path  string `json:"path"`  // base derivation path like "m/44'/0'/0'/0/0"
	Pk    string `json:"pk"`    // public key for coin
	Mode  string `json:"mode"`  // address version (public key)
	Addr  string `json:"addr"`  // address for base derivation path
}

func (c *CoinConfig) GetMode() uint32 {
	var val int64
	var err error
	if strings.HasPrefix(c.Mode, "0x") {
		val, err = strconv.ParseInt(c.Mode[2:], 16, 32)
	} else {
		val, err = strconv.ParseInt(c.Mode, 10, 32)
	}
	if err != nil {
		val = 0
	}
	return uint32(val)
}

// Config holds overall configuration settings
type Config struct {
	Coins []*CoinConfig `json:"coins"` // list of known coins
}

// ReadConfig to parse configurations from file
func ReadConfig(fname string) (*Config, error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	cfg := new(Config)
	if err = json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// WriteConfig to store configuration to file
func WriteConfig(fname string, cfg *Config) error {
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return err
}
