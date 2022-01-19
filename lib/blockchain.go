//----------------------------------------------------------------------
// This file is part of 'bitbank-relay'.
// Copyright (C) 2021 Bernd Fix >Y<
//
// 'bitbank-relay' is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
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
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/bfix/gospel/network"
)

// ChainHandler interface for blockchain-related processing
type ChainHandler interface {
	Init(cfg *HandlerConfig)
	Balance(addr string) (float64, error)
	GetFunds(ctx context.Context, addrId int64, addr string) ([]*Fund, error)
	Explore(addr string) string
	Limit() float64
}

// Instantiate a new blockchain handler based on coin symbol
func NewChainHandler(coin string, cfg *HandlerConfig) (hdlr ChainHandler) {
	switch coin {
	case "btc":
		hdlr = new(BtcChainHandler)
		hdlr.Init(cfg)
		return
	case "bch":
		hdlr = new(BchChainHandler)
		hdlr.Init(cfg)
		return
	case "btg":
		hdlr = new(BtgChainHandler)
		hdlr.Init(cfg)
		return
	case "dash":
		hdlr = new(DashChainHandler)
		hdlr.Init(cfg)
		return
	case "dgb":
		hdlr = new(DgbChainHandler)
		hdlr.Init(cfg)
		return
	case "doge":
		hdlr = new(DogeChainHandler)
		hdlr.Init(cfg)
		return
	case "ltc":
		hdlr = new(LtcChainHandler)
		hdlr.Init(cfg)
		return
	case "nmc":
		hdlr = new(NmcChainHandler)
		hdlr.Init(cfg)
		return
	case "vtc":
		hdlr = new(VtcChainHandler)
		hdlr.Init(cfg)
		return
	case "zec":
		hdlr = new(ZecChainHandler)
		hdlr.Init(cfg)
		return
	case "eth":
		hdlr = new(EthChainHandler)
		hdlr.Init(cfg)
		return
	case "etc":
		hdlr = new(EtcChainHandler)
		hdlr.Init(cfg)
		return
	}
	return nil
}

func ChainQuery(ctx context.Context, query string) ([]byte, error) {
	// time-out HTTP client
	toCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	cl := &http.Client{}

	// request information
	req, err := http.NewRequestWithContext(toCtx, http.MethodGet, query, nil)
	if err != nil {
		return nil, err
	}
	resp, err := cl.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// read and parse response
	return ioutil.ReadAll(resp.Body)
}

//======================================================================
// Shared blockchain handlers
//======================================================================

// singleton instances of shared folders
var (
	cciHandler = new(CCIChainHandler)
	bcHandler  = new(BcChainHandler)
)

// GenericChainHandler handles for multi-coin blockchain operations
type GenericChainHandler struct {
	limit    float64
	explorer string
}

// Init a new chain handler instance
func (hdlr *GenericChainHandler) Init(cfg *HandlerConfig) {
	hdlr.limit = cfg.Limit
	hdlr.explorer = cfg.Explorer
}

// GetFunds returns a list of incoming funds for the address
func (hdlr *GenericChainHandler) GetFunds(ctx context.Context, addrId int64, addr string) ([]*Fund, error) {
	return nil, nil
}

// Exporer returns the pattern for the blockchain browser URL
func (hdlr *GenericChainHandler) Explore(addr string) string {
	return hdlr.explorer
}

// Limit is the max. funding of an address (auto-close)
func (hdlr *GenericChainHandler) Limit() float64 {
	return hdlr.limit
}

//----------------------------------------------------------------------
// (chainz.cryptoid.info)
//----------------------------------------------------------------------

// CCIChainHandler handles multi-coin blockchain operations
type CCIChainHandler struct {
	ratelimiter *network.RateLimiter
	apiKey      string
	initialized bool // handler set-up?
}

// Init a new chain handler instance
func (hdlr *CCIChainHandler) Init(cfg *HandlerConfig) {
	// shared instance: init only once (first wins)
	if !hdlr.initialized {
		hdlr.initialized = true
		hdlr.ratelimiter = network.NewRateLimiter(cfg.Rates...)
		hdlr.apiKey = cfg.ApiKey
	}
}

// Balance gets the balance of a Bitcoin address
func (hdlr *CCIChainHandler) Balance(addr, coin string) (float64, error) {
	// perform query
	hdlr.ratelimiter.Pass()
	query := fmt.Sprintf("https://chainz.cryptoid.info/%s/api.dws?q=getbalance&a=%s", coin, addr)
	body, err := ChainQuery(context.Background(), query)
	if err != nil {
		return -1, err
	}
	val, err := strconv.ParseFloat(string(body), 64)
	if err != nil {
		return -1, err
	}
	return val, nil
}

// GetFunds returns a list of incoming funds for the address
func (hdlr *CCIChainHandler) GetFunds(ctx context.Context, addrId int64, addr string) ([]*Fund, error) {
	return nil, nil
}

//----------------------------------------------------------------------
// (blockchair.com)
//----------------------------------------------------------------------

// BcChainHandler handles multi-coin blockchain operations
type BcChainHandler struct {
	ratelimiter *network.RateLimiter
	limit       float64
	apiKey      string
	initialized bool // handler set-up?
}

// Init a new chain handler instance
func (hdlr *BcChainHandler) Init(cfg *HandlerConfig) {
	// shared instance: init only once (first wins)
	if !hdlr.initialized {
		hdlr.initialized = true
		hdlr.ratelimiter = network.NewRateLimiter(cfg.Rates...)
		hdlr.limit = cfg.Limit
		hdlr.apiKey = cfg.ApiKey
	}
}

// Balance gets the balance of a coin address
func (hdlr *BcChainHandler) Balance(addr, coin string) (float64, error) {
	// perform query
	hdlr.ratelimiter.Pass()
	query := fmt.Sprintf("https://api.blockchair.com/%s/dashboards/address/%s", coin, addr)
	if hdlr.apiKey != "" {
		query += fmt.Sprintf("?key=%s", hdlr.apiKey)
	}
	body, err := ChainQuery(context.Background(), query)
	if err != nil {
		return -1, err
	}
	// parse response
	data := new(BlockchairAddrInfo)
	if err = json.Unmarshal(body, &data); err != nil {
		return -1, err
	}
	// check status code.
	if data.Context.Code != 200 {
		return -1, fmt.Errorf("HTTP response %d", data.Context.Code)
	}
	// return response
	return float64(data.Data[addr].Address.Balance) / 1e8, nil
}

// GetFunds returns a list of incoming funds for the address
func (hdlr *BcChainHandler) GetFunds(ctx context.Context, addrId int64, addr string) ([]*Fund, error) {
	return nil, nil
}

// Limit is the max. funding of an address (auto-close)
func (hdlr *BcChainHandler) Limit() float64 {
	return hdlr.limit
}

// BlockchairAddrInfo is the response from the blockchair.com API
type BlockchairAddrInfo struct {
	Data map[string]struct {
		Address struct {
			Type               string                 `json:"type"`
			Script             string                 `json:"script_hex"`
			Balance            int64                  `json:"balance"`
			BalanceUSD         float64                `json:"balance_usd"`
			Received           float64                `json:"received"`
			ReceivedUSD        float64                `json:"received_usd"`
			Spent              float64                `json:"spent"`
			SpentUSD           float64                `json:"spent_usd"`
			OutputCount        int                    `json:"output_count"`
			UnspendOutputCount int                    `json:"unspent_output_count"`
			FirstSeenRecv      string                 `json:"first_seen_receiving"`
			LastSeenRecv       string                 `json:"last_seen_receiving"`
			FirstSeenSpending  string                 `json:"first_seen_spending"`
			LastSeenSpending   string                 `json:"last_seen_spending"`
			ScriptHashType     string                 `json:"scripthash_type"`
			TxCount            int                    `json:"transaction_count"`
			Formats            map[string]interface{} `json:"formats"`
		}
		Transactions []interface{} `json:"transactions"`
		UTXO         []interface{} `json:"utxo"`
	} `json:"data"`
	Context struct {
		Code    int    `json:"code"`
		Source  string `json:"source"`
		Results int    `json:"results"`
		State   int    `json:"state"`
		Cache   struct {
			Live     bool   `json:"live"`
			Duration int    `json:"duration"`
			Since    string `json:"since"`
			Until    string `json:"until"`
			Time     interface{}
		} `json:"cache"`
		API struct {
			Version       string `json:"version"`
			LastUpdate    string `json:"last_major_update"`
			NextUpdate    string `json:"next_major_update"`
			Documentation string `json:"documentation"`
			Notice        string `json:"notice"`
		} `json:"api"`
		Server      string  `json:"server"`
		Time        float64 `json:"time"`
		RenderTime  float64 `json:"render_time"`
		FulTime     float64 `json:"full_time"`
		RequestCost float64 `json:"request_cost"`
	} `json:"context"`
}
