//----------------------------------------------------------------------
// This file is part of 'bitbank-relay'.
// Copyright (C) 2021 Bernd Fix >Y<
//
// 'bitbank-relay' is free software: you can redistribute it and/or modify it
// under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// 'bitbank-relay' is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
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
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/bfix/gospel/bitcoin/wallet"
)

//----------------------------------------------------------------------

// CoinConfig for a supported coin (Bitcoin or Altcoin)
type CoinConfig struct {
	Symb       string  `json:"symb"`       // coin symbol
	Path       string  `json:"path"`       // base derivation path like "m/44'/0'/0'/0/0"
	Mode       string  `json:"mode"`       // address version (P2PKH, P2SH, ...)
	Pk         string  `json:"pk"`         // public key for coin
	Addr       string  `json:"addr"`       // address for base derivation path
	Limit      float64 `json:"limit"`      // limit for receiving addresses
	Explorer   string  `json:"explorer"`   // address explorer URL
	Blockchain string  `json:"blockchain"` // blockchain handler rerefence
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
	coin, _ := wallet.GetCoinInfo(c.Symb)
	if coin < 0 {
		return 0
	}
	return wallet.GetXDVersion(coin, m, wallet.AddrMain, true)
}

//----------------------------------------------------------------------

// ServiceConfig for service-related settings
type ServiceConfig struct {
	Listen    string `json:"listen"`    // web service listener (host:port)
	Epoch     int    `json:"epoch"`     // epoch time in seconds
	LogFile   string `json:"logFile"`   // logfile name
	LogLevel  string `json:"logLevel"`  // logging level
	LogRotate int    `json:"logRotate"` // epochs between log rotation
}

//----------------------------------------------------------------------

// ModelConfig for model-related settings.
type ModelConfig struct {
	DbEngine    string    `json:"dbEngine"`    // mode (mysql, sqlite3, ...)
	DbConnect   string    `json:"dbConnect"`   // database connect string
	BalanceWait []float64 `json:"balanceWait"` // wait parameters [min, factor, max]
	TxTTL       int       `json:"txTTL"`       // Time-to-live for Tx
}

//----------------------------------------------------------------------

// MarketHandlerConfig defines settings for cryptocurrency price retrieval.
type MarketHandlerConfig struct {
	RateLimits []int  `json:"rateLimits"` // rate limits
	ApiKey     string `json:"apikey"`     // authentication
}

// ChainHandlerConfig to sezup blockchain-retrieval handlers
type ChainHandlerConfig struct {
	RateLimits []int   `json:"rateLimits"` // rate limits
	CoolTime   float64 `json:"coolTime"`   // cool time between requests
	ApiKey     string  `json:"apiKey"`     // authentication
}

type MarketConfig struct {
	Fiat    string                          `json:"fiat"`    // Fiat base currency
	Rescan  int                             `json:"rescan"`  // rescan time interval (in epochs)
	Service map[string]*MarketHandlerConfig `json:"service"` // narket services
}

// HandlerConfig holds all handler-related configurations
type HandlerConfig struct {
	Blockchain map[string]*ChainHandlerConfig `json:"blockchain"`
	Market     *MarketConfig                  `json:"market"`
}

//----------------------------------------------------------------------

// Config holds overall configuration settings
type Config struct {
	Service *ServiceConfig `json:"service"` // web service configuration
	Model   *ModelConfig   `json:"model"`   // model configuration
	Handler *HandlerConfig `json:"handler"` // handler configuration
	Coins   []*CoinConfig  `json:"coins"`   // list of known coins
}

//----------------------------------------------------------------------
// persistent configuration

// ReadConfigFile parses a configuration from a file
func ReadConfigFile(fname string) (*Config, error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadConfig(f)
}

// ReadConfig to parse configurations from a reader
func ReadConfig(rdr io.Reader) (*Config, error) {
	data, err := ioutil.ReadAll(rdr)
	if err != nil {
		return nil, err
	}
	cfg := new(Config)
	if err = json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	//buf, _ := json.MarshalIndent(cfg, "", "  ")
	//logger.Println(logger.DBG, string(buf))
	return cfg, nil
}

// WriteConfigFile to store configuration to file
func WriteConfigFile(fname string, cfg *Config) error {
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteConfig(f, cfg)
}

// WriteConfig to write configuration to writer
func WriteConfig(wrt io.Writer, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		return err
	}
	_, err = wrt.Write(data)
	return err
}
