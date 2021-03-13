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
	"strings"

	"github.com/bfix/gospel/bitcoin/wallet"
)

// CoinConfig is a configuration for a supported coin (Bitcoin or Altcoin)
type CoinConfig struct {
	Name  string `json:"name"`  // coin symbol
	Descr string `json:"descr"` // coin description
	Path  string `json:"path"`  // base derivation path like "m/44'/0'/0'/0/0"
	Pk    string `json:"pk"`    // public key for coin
	Mode  string `json:"mode"`  // address version (P2PKH, P2SH, ...)
	Addr  string `json:"addr"`  // address for base derivation path
}

// GetMode returns the numeric value of mode (P2PKH, P2SH, ...)
func (c *CoinConfig) GetMode() int {
	switch strings.ToUpper(c.Mode) {
	case "P2PKH":
		return wallet.AddrP2PKH
	case "P2SH":
		return wallet.AddrP2SH
	case "P2WPKH":
		return wallet.AddrP2WPKH
	case "P2WSH":
		return wallet.AddrP2WSH
	case "P2WPKHinP2SH":
		return wallet.AddrP2WPKHinP2SH
	case "P2WSHinP2SH":
		return wallet.AddrP2WSHinP2SH
	}
	return -1
}

// GetXDVersion returns the extended data version for coin
func (c *CoinConfig) GetXDVersion() uint32 {
	m := c.GetMode()
	if m < 0 {
		return wallet.XpubVersion
	}
	coin := wallet.GetCoinID(c.Name)
	if coin < 0 {
		return 0
	}
	return wallet.GetXDVersion(coin, m, wallet.AddrMain, true)
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

// GetNetwork returns the numeric coin network ID
func GetNetwork(netw string) int {
	switch strings.ToLower(netw) {
	case "main":
		return wallet.AddrMain
	case "test":
		return wallet.AddrTest
	case "reg":
		return wallet.AddrReg
	}
	return -1
}
