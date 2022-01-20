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
	"sync"
	"time"

	"github.com/bfix/gospel/network"
)

//----------------------------------------------------------------------
// Chain handlers: All external data (like balances and transactions
// stored on a blockchain) for addresses/coins is managed by a chain
// handler instance for a coins.
//----------------------------------------------------------------------

// ChainHandler interface for blockchain-related processing
type ChainHandler interface {
	Init(cfg *HandlerConfig)
	Balance(addr string) (float64, error)
	GetFunds(ctx context.Context, addrId int64, addr string) ([]*Fund, error)
	Explore(addr string) string
	Limit() float64
}

// SharedChainHandler interface for multi-coin chain handlers
type SharedChainHandler interface {
	Init(cfg *HandlerConfig)
	Balance(addr, coin string) (float64, error)
	GetFunds(ctx context.Context, addrId int64, addr, coin string) ([]*Fund, error)
}

//----------------------------------------------------------------------
// A derived chain handler manages a single coin by using a shared
// chain handler for its operations.
//----------------------------------------------------------------------

// DerivedChainHandler manages a single coin with a shared handler
type DerivedChainHandler struct {
	coin     string             // associated coin symbol
	parent   SharedChainHandler // reference to parent handler
	limit    float64            // account limit (auto-closing)
	explorer string             // URL pattern for blockchain browser
}

// Init a new chain handler instance
func (hdlr *DerivedChainHandler) Init(cfg *HandlerConfig) {
	hdlr.limit = cfg.Limit
	hdlr.explorer = cfg.Explorer
}

// Balance gets the balance of an address
func (hdlr *DerivedChainHandler) Balance(addr string) (float64, error) {
	return hdlr.parent.Balance(addr, hdlr.coin)
}

// GetFunds returns a list of incoming funds for the address
func (hdlr *DerivedChainHandler) GetFunds(ctx context.Context, addrId int64, addr string) ([]*Fund, error) {
	return nil, nil
}

// Exporer returns the pattern for the blockchain browser URL
func (hdlr *DerivedChainHandler) Explore(addr string) string {
	return hdlr.explorer
}

// Limit is the max. funding of an address (auto-close)
func (hdlr *DerivedChainHandler) Limit() float64 {
	return hdlr.limit
}

//----------------------------------------------------------------------
// Basic chain handlers are generic stand-alone handlers for a coin
//----------------------------------------------------------------------

// BtcChainHandler handles BTC-related blockchain operations
type BasicChainHandler struct {
	ratelimiter *network.RateLimiter
	limit       float64
	apiKey      string
	explorer    string
	lock        sync.Mutex
}

// Init a new chain handler instance
func (hdlr *BasicChainHandler) Init(cfg *HandlerConfig) {
	hdlr.ratelimiter = network.NewRateLimiter(cfg.Rates...)
	hdlr.limit = cfg.Limit
	hdlr.apiKey = cfg.ApiKey
	hdlr.explorer = cfg.Explorer
}

// Exporer returns the pattern for the blockchain browser URL
func (hdlr *BasicChainHandler) Explore(addr string) string {
	return hdlr.explorer
}

// Limit is the max. funding of an address (auto-close)
func (hdlr *BasicChainHandler) Limit() float64 {
	return hdlr.limit
}

//======================================================================
// Shared blockchain handlers
//======================================================================

// singleton instances of shared handlers
var (
	cciHandler = new(CCIChainHandler)
	bcHandler  = new(BcChainHandler)
)

//----------------------------------------------------------------------
// (chainz.cryptoid.info)
//----------------------------------------------------------------------

// CCIChainHandler handles multi-coin blockchain operations
type CCIChainHandler struct {
	lastCall    int64      // time last used (UnixMilli)
	apiKey      string     // optional API key
	initialized bool       // handler set-up?
	lock        sync.Mutex // serialize operations
}

// wait for execution of request: requests are serialized and
func (hdlr *CCIChainHandler) wait() {
	// only handle one call at a time
	hdlr.lock.Lock()
	defer hdlr.lock.Unlock()

	delay := time.Now().UnixMilli() - hdlr.lastCall
	if delay < 10000 {
		time.Sleep(time.Duration(10000-delay) * time.Millisecond)
	}
	hdlr.lastCall = time.Now().UnixMilli()
}

// Init a new chain handler instance
func (hdlr *CCIChainHandler) Init(cfg *HandlerConfig) {
	// shared instance: init only once (first wins)
	if !hdlr.initialized {
		hdlr.initialized = true
		hdlr.apiKey = cfg.ApiKey
	}
}

// Balance gets the balance of a Bitcoin address
func (hdlr *CCIChainHandler) Balance(addr, coin string) (float64, error) {
	// perform query
	hdlr.wait()
	query := fmt.Sprintf("https://chainz.cryptoid.info/%s/api.dws?q=getreceivedbyaddress&a=%s", coin, addr)
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
func (hdlr *CCIChainHandler) GetFunds(ctx context.Context, addrId int64, addr, coin string) ([]*Fund, error) {
	// only handle one call at a time
	return nil, nil
}

//----------------------------------------------------------------------
// (blockchair.com)
//----------------------------------------------------------------------

// BcChainHandler handles multi-coin blockchain operations
type BcChainHandler struct {
	ratelimiter *network.RateLimiter // limit calls to service
	apiKey      string               // optional API key
	initialized bool                 // handler set-up?
	lock        sync.Mutex           // serialize operations
}

// Init a new chain handler instance
func (hdlr *BcChainHandler) Init(cfg *HandlerConfig) {
	// shared instance: init only once (first wins)
	if !hdlr.initialized {
		hdlr.initialized = true
		hdlr.ratelimiter = network.NewRateLimiter(cfg.Rates...)
		hdlr.apiKey = cfg.ApiKey
	}
}

// query address information (incl. transaction list)
func (hdlr *BcChainHandler) query(addr, coin string) (*BlockchairAddrInfo, error) {
	// only handle one call at a time
	hdlr.lock.Lock()
	defer hdlr.lock.Unlock()

	// perform query
	hdlr.ratelimiter.Pass()
	query := fmt.Sprintf("https://api.blockchair.com/%s/dashboards/address/%s", coin, addr)
	if hdlr.apiKey != "" {
		query += fmt.Sprintf("?key=%s", hdlr.apiKey)
	}
	body, err := ChainQuery(context.Background(), query)
	if err != nil {
		return nil, err
	}
	// parse response
	data := new(BlockchairAddrInfo)
	if err = json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	// check status code.
	if data.Context.Code != 200 {
		return nil, fmt.Errorf("HTTP response %d", data.Context.Code)
	}
	return data, nil
}

// Balance gets the balance of a coin address
func (hdlr *BcChainHandler) Balance(addr, coin string) (float64, error) {
	// get address information
	data, err := hdlr.query(addr, coin)
	if err != nil {
		return -1, err
	}
	// return response
	return float64(data.Data[addr].Address.Received) / 1e8, nil
}

// GetFunds returns a list of incoming funds for the address
func (hdlr *BcChainHandler) GetFunds(ctx context.Context, addrId int64, addr, coin string) ([]*Fund, error) {
	// get address information
	data, err := hdlr.query(addr, coin)
	if err != nil {
		return nil, err
	}
	// collect funding transactions
	funds := make([]*Fund, 0)
	for _, txHash := range data.Data[addr].Transactions {
		// perform query
		hdlr.ratelimiter.Pass()
		query := fmt.Sprintf("https://api.blockchair.com/%s/dashboards/transaction/%s", coin, txHash)
		if hdlr.apiKey != "" {
			query += fmt.Sprintf("?key=%s", hdlr.apiKey)
		}
		body, err := ChainQuery(context.Background(), query)
		if err != nil {
			return nil, err
		}
		// parse response
		rec := new(BlockchairTxInfo)
		if err = json.Unmarshal(body, &rec); err != nil {
			return nil, err
		}
		tx := rec.Data[txHash]
		// find received funds in transaction outputs
		for _, vout := range tx.Outputs {
			if addr == vout.Recipient {
				ts, err := time.Parse("2006-02-01 15:04:05", vout.Time)
				if err != nil {
					return nil, err
				}
				f := &Fund{
					Seen:   ts.Unix(),
					Addr:   addrId,
					Amount: float64(vout.Value) / 1e8,
				}
				funds = append(funds, f)
			}
		}
	}
	return funds, nil
}

// BlockChairContext for the API request
type BlockChairContext struct {
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
		Transactions []string `json:"transactions"`
		UTXO         []*struct {
			BlockId int    `json:"block_id"`
			TxHash  string `json:"transaction_hash"`
			Index   int    `json:"index"`
			Value   int64  `json:"value"`
		} `json:"utxo"`
	} `json:"data"`
	Context *BlockChairContext `json:"context"`
}

// BlockchairTxSlot is an input/output slot of the transaction
type BlockchairTxSlot struct {
	BlockId          int     `json:"block_id"`
	TxId             int64   `json:"transaction_id"`
	Index            int     `json:"index"`
	TxHash           string  `json:"transaction_hash"`
	Date             string  `json:"date"`
	Time             string  `json:"time"`
	Value            int64   `json:"value"`
	ValueUSD         int64   `json:"value_usd"`
	Recipient        string  `json:"recipient"`
	Type             string  `json:"type"`
	ScriptHex        string  `json:"script_hex"`
	FromCoinbase     bool    `json:"is_from_coinbase"`
	IsSpendable      *bool   `json:"is_spendable"`
	IsSpent          bool    `json:"is_spent"`
	SpendingBlkId    int     `json:"spending_block_id"`
	SpendingTxId     int64   `json:"spending_transaction_id"`
	SpendingIndex    int     `json:"spending_index"`
	SpendingTxHash   string  `json:"spending_transaction_hash"`
	SpendingDate     string  `json:"spending_date"`
	SpendingTime     string  `json:"spending_time"`
	SpendingValueUDS int64   `json:"spending_value_usd"`
	SpendingSequence int64   `json:"spending_sequence"`
	SpendingSigHex   string  `json:"spending_signature_hex"`
	LifeSpan         int64   `json:"lifespan"`
	Cdd              float64 `json:"cdd"`
}

// BlockchairTxInfo is a transaction response
type BlockchairTxInfo struct {
	Data map[string]struct {
		Transaction struct {
			BlockId     int     `json:"block_id"`
			Id          int     `json:"id"`
			Hash        string  `json:"hash"`
			Date        string  `json:"date"`
			Time        string  `json:"time"`
			Size        int     `json:"size"`
			Version     int     `json:"version"`
			LockTime    int64   `json:"lock_time"`
			IsCoinbase  bool    `json:"is_coinbase"`
			InCount     int     `json:"input_count"`
			OutCount    int     `json:"output_count"`
			InTotal     int64   `json:"input_total"`
			InTotalUSD  int64   `json:"input_total_usd"`
			OutTotal    int64   `json:"output_total"`
			OutTotalUSD int64   `json:"output_total_usd"`
			Fee         int64   `json:"fee"`
			FeeUSD      float64 `json:"fee_usd"`
			FeeKB       float64 `json:"fee_per_kb"`
			FeeKBUSD    float64 `json:"fee_per_kb_usd"`
			CddTotal    float64 `json:"cdd_total"`
		} `json:"transaction"`
		Inputs  []*BlockchairTxSlot `json:"inputs"`
		Outputs []*BlockchairTxSlot `json:"outputs"`
		Context *BlockChairContext  `json:"context"`
	} `json:"data"`
}

//----------------------------------------------------------------------
// Instantiation of chain handler instances
//----------------------------------------------------------------------

var (
	chainHdlr = map[string]ChainHandler{
		"btc":  new(BtcChainHandler),
		"bch":  new(BchChainHandler),
		"btg":  new(BtgChainHandler),
		"dash": new(DashChainHandler),
		"dgb":  new(DgbChainHandler),
		"doge": new(DogeChainHandler),
		"ltc":  new(LtcChainHandler),
		"nmc":  new(NmcChainHandler),
		"vtc":  new(VtcChainHandler),
		"zec":  new(ZecChainHandler),
		"eth":  new(EthChainHandler),
		"etc":  new(EtcChainHandler),
	}
)

// Instantiate a new blockchain handler based on coin symbol
func NewChainHandler(coin string, cfg *HandlerConfig) (hdlr ChainHandler) {
	hdlr, ok := chainHdlr[coin]
	if ok {
		hdlr.Init(cfg)
	} else {
		hdlr = nil
	}
	return
}

//----------------------------------------------------------------------
// Helper functions
//----------------------------------------------------------------------

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
