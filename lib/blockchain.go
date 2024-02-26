//----------------------------------------------------------------------
// This file is part of 'bitbank-relay'.
// Copyright (C) 2021-2024, Bernd Fix >Y<
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
	"io"
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
	Init(cfg *ChainHandlerConfig)
	Balance(ctx context.Context, addr, coin string) (float64, error)
	GetFunds(ctx context.Context, addrId int64, addr, coin string) ([]*Fund, error)
}

//----------------------------------------------------------------------
// Basic chain handlers are generic stand-alone handlers for a coin
//----------------------------------------------------------------------

// BasicChainHandler handles BTC-related blockchain operations
type BasicChainHandler struct {
	ratelimiter *network.RateLimiter
	apiKey      string
	lock        sync.Mutex
}

// Init a new chain handler instance
func (hdlr *BasicChainHandler) Init(cfg *ChainHandlerConfig) {
	hdlr.ratelimiter = network.NewRateLimiter(cfg.RateLimits...)
	hdlr.apiKey = cfg.ApiKey
}

//======================================================================
// Shared blockchain handlers
//======================================================================

// singleton instances of shared handlers
var (
	baseChainHdlrs = map[string]ChainHandler{
		"cryptoid.info":   new(CciChainHandler),
		"blockchair.com":  new(BcChainHandler),
		"btgexplorer.com": new(BtgChainHandler),
		"zcha.in":         new(ZecChainHandler),
		"blockscout.com":  new(EtcChainHandler),
	}
)

//----------------------------------------------------------------------
// (chainz.cryptoid.info)
//----------------------------------------------------------------------

// CciChainHandler handles multi-coin blockchain operations
type CciChainHandler struct {
	lastCall    int64      // time last used (UnixMilli)
	coolTime    float64    // time between calls
	apiKey      string     // optional API key
	initialized bool       // handler set-up?
	lock        sync.Mutex // serialize operations
}

// wait for execution of request: requests are serialized and
func (hdlr *CciChainHandler) wait(withLock bool) {
	// only handle one call at a time
	if withLock {
		hdlr.lock.Lock()
		defer hdlr.lock.Unlock()
	}

	delay := time.Now().UnixMilli() - hdlr.lastCall
	bounds := int64(hdlr.coolTime * 1000)
	if delay < bounds {
		time.Sleep(time.Duration(bounds-delay) * time.Millisecond)
	}
	hdlr.lastCall = time.Now().UnixMilli()
}

// Init a new chain handler instance
func (hdlr *CciChainHandler) Init(cfg *ChainHandlerConfig) {
	// shared instance: init only once (first wins)
	if !hdlr.initialized {
		hdlr.initialized = true
		hdlr.apiKey = cfg.ApiKey
		hdlr.coolTime = cfg.CoolTime
	}
}

// Balance gets the balance of a Bitcoin address
func (hdlr *CciChainHandler) Balance(ctx context.Context, addr, coin string) (float64, error) {
	// perform query
	hdlr.wait(true)
	query := fmt.Sprintf("https://chainz.cryptoid.info/%s/api.dws?q=getreceivedbyaddress&a=%s", coin, addr)
	if hdlr.apiKey != "" {
		query += fmt.Sprintf("&key=%s", hdlr.apiKey)
	}
	body, err := HTTPQuery(ctx, query)
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
func (hdlr *CciChainHandler) GetFunds(ctx context.Context, addrId int64, addr, coin string) ([]*Fund, error) {
	// perform query
	hdlr.wait(true)
	query := fmt.Sprintf("https://chainz.cryptoid.info/%s/api.dws?q=multiaddr&active=%s", coin, addr)
	if hdlr.apiKey != "" {
		query += fmt.Sprintf("&key=%s", hdlr.apiKey)
	}
	body, err := HTTPQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	// parse response
	data := new(CciAddrInfo)
	if err = json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	// collect funding transactions
	funds := make([]*Fund, 0)
	for _, tx := range data.Txs {
		// query transaction
		hdlr.wait(false)
		query := fmt.Sprintf("https://chainz.cryptoid.info/%s/api.dws?q=txinfo&t=%s", coin, tx.Hash)
		if hdlr.apiKey != "" {
			query += fmt.Sprintf("?key=%s", hdlr.apiKey)
		}
		if body, err = HTTPQuery(context.Background(), query); err != nil {
			return nil, err
		}
		// parse response
		tx := new(CciTxInfo)
		if err = json.Unmarshal(body, &tx); err != nil {
			return nil, err
		}
		// find received funds in transaction outputs
		for _, vout := range tx.Outputs {
			if addr == vout.Addr {
				f := &Fund{
					Seen:   tx.Timestamp,
					Addr:   addrId,
					Amount: vout.Amount,
				}
				funds = append(funds, f)
			}
		}
	}
	return funds, nil

}

// CciAddrInfo holds basic address information
type CciAddrInfo struct {
	Addresses []struct {
		Address       string `json:"address"`
		TotalSent     int64  `json:"total_sent"`
		TotalReceived int64  `json:"total_received"`
		FinalBalance  int64  `json:"final_balance"`
		NTx           int    `json:"n_tx"`
	} `json:"addresses"`
	Txs []struct {
		Hash          string `json:"hash"`
		Confirmations int    `json:"confirmations"`
		Change        int64  `json:"change"`
		TimeUTC       string `json:"time_utc"`
	} `json:"txs"`
}

// CciTxInfo holds transaction details
type CciTxInfo struct {
	Hash          string  `json:"hash"`
	Block         int     `json:"block"`
	Index         int     `json:"index"`
	Timestamp     int64   `json:"timestamp"`
	Confirmations int     `json:"confirmations"`
	Fees          float64 `json:"fees"`
	TotalInput    float64 `json:"total_input"`
	Inputs        []struct {
		Addr         string  `json:"addr"`
		Amount       float64 `json:"amount"`
		ReceivedFrom struct {
			Tx string `json:"tx"`
			N  int    `json:"n"`
		} `json:"received_from"`
	} `json:"inputs"`
	TotalOutputs float64 `json:"total_output"`
	Outputs      []struct {
		Addr   string  `json:"addr"`
		Amount float64 `json:"amount"`
		Script string  `json:"script"`
	} `json:"outputs"`
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
func (hdlr *BcChainHandler) Init(cfg *ChainHandlerConfig) {
	// shared instance: init only once (first wins)
	if !hdlr.initialized {
		hdlr.initialized = true
		hdlr.ratelimiter = network.NewRateLimiter(cfg.RateLimits...)
		hdlr.apiKey = cfg.ApiKey
	}
}

var (
	// map coin ticker into coin name used by handler instance
	bcCoinMap = map[string]string{
		"btc":  "bitcoin",
		"bch":  "bitcoin-cash",
		"dash": "dash",
		"doge": "dogecoin",
		"ltc":  "litecoin",
		"eth":  "ethereum",
	}
	// map coin ticker into scale used by handler instance
	bcScaleMap = map[string]float64{
		"btc":  1e8,
		"bch":  1e8,
		"dash": 1e8,
		"doge": 1e8,
		"ltc":  1e8,
		"eth":  1e18,
	}
)

// query address information (incl. transaction list)
func (hdlr *BcChainHandler) query(ctx context.Context, addr, coin string) (*BlockchairAddrInfo, error) {
	// only handle one call at a time
	hdlr.lock.Lock()
	defer hdlr.lock.Unlock()

	// perform query
	hdlr.ratelimiter.Pass()
	c, ok := bcCoinMap[coin]
	if !ok {
		c = coin
	}
	query := fmt.Sprintf("https://api.blockchair.com/%s/dashboards/address/%s", c, addr)
	if hdlr.apiKey != "" {
		query += fmt.Sprintf("?key=%s", hdlr.apiKey)
	}
	body, err := HTTPQuery(ctx, query)
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
func (hdlr *BcChainHandler) Balance(ctx context.Context, addr, coin string) (float64, error) {
	// get address information
	data, err := hdlr.query(ctx, addr, coin)
	if err != nil {
		return -1, err
	}
	// return response
	ai := data.Data[addr].Address
	rcv := ai.Received
	if len(ai.ReceivedApprox) > 0 {
		rcv, err = strconv.ParseFloat(ai.ReceivedApprox, 64)
		if err != nil {
			return -1, err
		}
	}
	return rcv / bcScaleMap[coin], nil
}

// GetFunds returns a list of incoming funds for the address
func (hdlr *BcChainHandler) GetFunds(ctx context.Context, addrId int64, addr, coin string) ([]*Fund, error) {
	// get address information
	data, err := hdlr.query(ctx, addr, coin)
	if err != nil {
		return nil, err
	}
	// map coin name to name used by handler
	c, ok := bcCoinMap[coin]
	if !ok {
		c = coin
	}
	// collect funding transactions
	funds := make([]*Fund, 0)
	for _, txHash := range data.Data[addr].Transactions {
		// perform query
		hdlr.ratelimiter.Pass()
		query := fmt.Sprintf("https://api.blockchair.com/%s/dashboards/transaction/%s", c, txHash)
		if hdlr.apiKey != "" {
			query += fmt.Sprintf("?key=%s", hdlr.apiKey)
		}
		body, err := HTTPQuery(ctx, query)
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
				ts, err := time.Parse("2006-01-02 15:04:05", vout.Time)
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
			Balance            interface{}            `json:"balance"`
			BalanceUSD         float64                `json:"balance_usd"`
			Received           float64                `json:"received"`
			ReceivedApprox     string                 `json:"received_approximate"`
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
	ValueUSD         float64 `json:"value_usd"`
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
	SpendingValueUSD float64 `json:"spending_value_usd"`
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
			InTotalUSD  float64 `json:"input_total_usd"`
			OutTotal    int64   `json:"output_total"`
			OutTotalUSD float64 `json:"output_total_usd"`
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

//======================================================================
// BTG (Bitcoin Gold)
//======================================================================

// BtgChainHandler handles BitcoinGold-related blockchain operations
type BtgChainHandler struct {
	BasicChainHandler
}

// Balance gets the balance of a Bitcoin Gold address
func (hdlr *BtgChainHandler) Balance(ctx context.Context, addr, coin string) (float64, error) {
	// only handle one call at a time
	hdlr.lock.Lock()
	defer hdlr.lock.Unlock()

	// perform query
	hdlr.ratelimiter.Pass()
	query := fmt.Sprintf("https://btgexplorer.com/api/address/%s", addr)
	body, err := HTTPQuery(ctx, query)
	if err != nil {
		return -1, err
	}
	data := new(BtgAddrInfo)
	if err = json.Unmarshal(body, &data); err != nil {
		return -1, err
	}
	// return balance (incoming funds)
	val, err := strconv.ParseFloat(data.TotalReceived, 64)
	if err != nil {
		return -1, err
	}
	// return balance
	return val, nil
}

// GetFunds returns incoming transaction for a Bitcoin Gold address.
func (hdlr *BtgChainHandler) GetFunds(ctx context.Context, addrId int64, addr, coin string) ([]*Fund, error) {
	// only handle one call at a time
	hdlr.lock.Lock()
	defer hdlr.lock.Unlock()

	// perform query (stage 1)
	hdlr.ratelimiter.Pass()
	query := fmt.Sprintf("https://btgexplorer.com/api/address/%s", addr)
	body, err := HTTPQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	data := new(BtgAddrInfo)
	if err = json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	// process all transactions
	funds := make([]*Fund, 0)
	for _, tx := range data.Transaction {
		// perform query (stage 2)
		hdlr.ratelimiter.Pass()
		query := fmt.Sprintf("https://btgexplorer.com/api/tx/%s", tx)
		body, err := HTTPQuery(ctx, query)
		if err != nil {
			continue
		}
		data := make([]*BtgTxInfo, 0)
		if err = json.Unmarshal(body, &data); err != nil {
			return nil, err
		}
		// find received funds in transaction outputs
		for _, tx := range data {
			for _, vout := range tx.Vout {
				val, err := strconv.ParseFloat(vout.Value, 64)
				if err != nil {
					continue
				}
				for _, a := range vout.ScriptPubKey.Addresses {
					if addr == a {
						f := &Fund{
							Seen:   tx.Time,
							Addr:   addrId,
							Amount: val,
						}
						funds = append(funds, f)
					}
				}
			}
		}
	}
	// return funds
	return funds, nil
}

//----------------------------------------------------------------------
// internal access helpers
//----------------------------------------------------------------------

// BtgAddrInfo is the response from the btgexplorer.com API
type BtgAddrInfo struct {
	Page               int      `json:"page"`
	TotalPages         int      `json:"totalPages"`
	ItemsOnPage        int      `json:"itemsOnPage"`
	Address            string   `json:"addrStr"`
	Balance            string   `json:"balance"`
	TotalReceived      string   `json:"totalReceived"`
	TotalSent          string   `json:"totalSent"`
	UnconfirmedBalance string   `json:"unconfirmedBalance"`
	UnconfirmedTxs     int      `json:"unconfirmedTxApperances"`
	TxApperances       int      `json:"txApperances"`
	Transaction        []string `json:"transactions"`
}

// BtgTxInfo represents a Bitcoin Gold transaction
type BtgTxInfo struct {
	TxID          string       `json:"txid"`
	Version       int          `json:"version"`
	Vin           []*BtgTxVin  `json:"vin"`
	Vout          []*BtgTxVout `json:"vout"`
	BlockHash     string       `json:"blockHash"`
	BlockHeight   int          `json:"blockHeight"`
	Confirmations int          `json:"confirmations"`
	Time          int64        `json:"time"`
	BlockTime     int64        `json:"blockTime"`
	ValueOut      float64      `json:"valueOut"`
	ValueIn       float64      `json:"valueIn"`
	Fee           string       `json:"fee"`
	Hex           string       `json:"hex"`
}

// BtgTxVin is an input slot
type BtgTxVin struct {
	TxID      string `json:"txid"`
	Vout      int    `json:"vout"`
	Sequence  int32  `json:"sequence"`
	N         int    `json:"n"`
	ScriptSig struct {
		Asm string `json:"asm"`
		Hex string `json:"hex"`
	} `json:"scriptSig"`
	Addresses []string `json:"addresses"`
	Value     string   `json:"value"`
}

// BtgTxVout is an output slot
type BtgTxVout struct {
	Value        string `json:"value"`
	N            int    `json:"n"`
	ScriptPubKey struct {
		Addresses []string `json:"addresses"`
		Asm       string   `json:"asm"`
		Hex       string   `json:"hex"`
	} `json:"scriptPubKey"`
	Spent bool `json:"spent"`
}

//======================================================================
// ETC (Ethereum Classic)
//======================================================================

// EtcChainHandler handles Ethereum Classic-related blockchain operations
type EtcChainHandler struct {
	BasicChainHandler
}

// Balance gets the balance of an Ethereum address
func (hdlr *EtcChainHandler) Balance(ctx context.Context, addr, coin string) (float64, error) {
	// only handle one call at a time
	hdlr.lock.Lock()
	defer hdlr.lock.Unlock()

	// perform query
	hdlr.ratelimiter.Pass()
	query := fmt.Sprintf("https://blockscout.com/etc/mainnet/api?module=account&action=balance&address=%s", addr)
	body, err := HTTPQuery(ctx, query)
	if err != nil {
		return -1, err
	}
	data := new(EtcAddrInfo)
	if err = json.Unmarshal(body, &data); err != nil {
		return -1, err
	}
	// return balance (incoming funds)
	if data.Result == nil {
		return -1, err
	}
	val, err := strconv.ParseInt(*data.Result, 10, 64)
	if err != nil {
		return -1, err
	}
	return float64(val) / 1e18, nil
}

// GetFunds returns incoming transaction for an Ethereum address.
func (hdlr *EtcChainHandler) GetFunds(ctx context.Context, addrId int64, addr, coin string) ([]*Fund, error) {
	// only handle one call at a time
	hdlr.lock.Lock()
	defer hdlr.lock.Unlock()

	// perform query
	hdlr.ratelimiter.Pass()
	query := fmt.Sprintf("https://blockscout.com/etc/mainnet/api?module=account&action=txlist&address=%s", addr)
	body, err := HTTPQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	data := new(EtcTxInfo)
	if err = json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	// find received funds in transaction outputs
	funds := make([]*Fund, 0)
	for _, tx := range data.Result {
		ts, err := strconv.ParseInt(tx.Timestamp, 10, 64)
		if err != nil {
			continue
		}
		val, err := strconv.ParseInt(tx.Value, 10, 64)
		if err != nil {
			continue
		}
		f := &Fund{
			Seen:   ts,
			Addr:   addrId,
			Amount: float64(val) / 1e18,
		}
		funds = append(funds, f)
	}
	// return funds
	return funds, nil
}

// EtcAddrInfo is a response for an address info query
type EtcAddrInfo struct {
	Message string  `json:"message"`
	Result  *string `json:"result"`
	Status  string  `json:"status"`
}

// EtcTxInfo is a response for an address transaction query
type EtcTxInfo struct {
	Message string `json:"message"`
	Result  []*struct {
		BlockHash       string `json:"blockHash"`
		BlockNumber     string `json:"blockNumber"`
		Confirmations   string `json:"confirmations"`
		ContractAddress string `json:"contractAddress"`
		CumGasUsed      string `json:"cumulativeGasUsed"`
		From            string `json:"from"`
		Gas             string `json:"gas"`
		GasPrice        string `json:"gasPrice"`
		GasedUsed       string `json:"gasUsed"`
		Hash            string `json:"hash"`
		Input           string `json:"input"`
		IsError         string `json:"isError"`
		None            string `json:"nonce"`
		Timestamp       string `jspn:"timeStamp"`
		To              string `json:"to"`
		TxIndex         string `json:"transactionIndex"`
		TxReceipt       string `json:"txreceipt_status"`
		Value           string `json:"value"`
	} `json:"result"`
	Status string `json:"status"`
}

//======================================================================
// ZEC (ZCash)
//======================================================================

// / ZecChainHandler handles ZCash-related blockchain operations
type ZecChainHandler struct {
	BasicChainHandler
}

// Balance gets the balance of a ZCash address
func (hdlr *ZecChainHandler) Balance(ctx context.Context, addr, coin string) (float64, error) {
	// only handle one call at a time
	hdlr.lock.Lock()
	defer hdlr.lock.Unlock()

	// assemble query
	hdlr.ratelimiter.Pass()
	query := fmt.Sprintf("https://api.zcha.in/v2/mainnet/accounts/%s", addr)
	body, err := HTTPQuery(ctx, query)
	if err != nil {
		return -1, err
	}
	data := new(ZecAddrInfo)
	if err = json.Unmarshal(body, &data); err != nil {
		return -1, err
	}
	// return balance
	return data.TotalRecv, nil
}

// GetFunds returns incoming transaction for a ZCash address.
func (hdlr *ZecChainHandler) GetFunds(ctx context.Context, addrId int64, addr, coin string) ([]*Fund, error) {
	// only handle one call at a time
	hdlr.lock.Lock()
	defer hdlr.lock.Unlock()

	// retrieve list of transactions in chunks
	funds := make([]*Fund, 0)
	offset := 0
	for {
		// perform query
		hdlr.ratelimiter.Pass()
		query := fmt.Sprintf(
			"https://api.zcha.in/v2/mainnet/accounts/%s/recv"+
				"?limit=20&offset=%d&sort=timestamp&direction=ascending",
			addr, offset)
		body, err := HTTPQuery(ctx, query)
		if err != nil {
			return nil, err
		}
		data := make([]*ZecAddrTx, 0)
		if err = json.Unmarshal(body, &data); err != nil {
			return nil, err
		}
		// find received funds in transaction outputs
		for _, tx := range data {
			for _, vout := range tx.Vout {
				for _, a := range vout.ScriptPubKey.Addresses {
					if addr == a {
						f := &Fund{
							Seen:   tx.Timestamp,
							Addr:   addrId,
							Amount: tx.Value,
						}
						funds = append(funds, f)
					}
				}
			}
		}
		// address next chunk
		n := len(data)
		if n < 20 {
			break
		}
		offset += n
	}
	// return funds
	return funds, nil
}

//----------------------------------------------------------------------
// internal access helpers
//----------------------------------------------------------------------

// ZecAddrInfo is a response from the zcha.in API for an address query
type ZecAddrInfo struct {
	Address    string  `json:"address"`
	Balance    float64 `json:"balance"`
	FirstSeen  int64   `json:"firstSeen"`
	LastSeen   int64   `json:"lastSeen"`
	SentCount  int     `json:"sentCount"`
	RecvCount  int     `json:"recvCount"`
	MinedCount int     `json:"minedCount"`
	TotalSent  float64 `json:"totalSent"`
	TotalRecv  float64 `json:"totalRecv"`
}

// ZecAddrTx represents a ZCash transaction
type ZecAddrTx struct {
	Hash            string        `json:"hash"`
	MainChain       bool          `json:"mainChain"`
	Fee             float64       `json:"fee"`
	Type            string        `json:"type"`
	Shielded        bool          `json:"shielded"`
	Index           int           `json:"index"`
	BlockHash       string        `json:"blockHash"`
	BlockHeight     int           `json:"blockHeight"`
	Version         int           `json:"version"`
	LockTime        int64         `json:"lockTime"`
	Timestamp       int64         `json:"timestamp"`
	Time            int           `json:"time"`
	Vin             []*ZecTxVin   `json:"vin"`
	Vout            []*ZecTxVout  `json:"vout"`
	VJoinSplit      []interface{} `json:"vjoinsplit"`
	VShieldedOutput float64       `json:"vShieldedOutput"`
	VShieldedSpend  float64       `json:"vShieldedSpend"`
	ValueBalance    float64       `json:"valueBalance"`
	Value           float64       `json:"value"`
	OutputValue     float64       `json:"outputValue"`
	ShieldedValue   float64       `json:"shieldedValue"`
	OverWintered    bool          `json:"overwintered"`
}

// ZecTxVin is an input slot
type ZecTxVin struct {
	Coinbase  string     `json:"coinbase"`
	RetrVOut  *ZecTxVout `json:"retrievedVout"`
	ScriptSig struct {
		Asm string `json:"asm"`
		Hex string `json:"hex"`
	} `json:"scriptSig"`
	Sequence int32  `json:"sequence"`
	TxID     string `json:"txid"`
	Vout     int    `json:"vout"`
}

// ZecTxVout is an output slot
type ZecTxVout struct {
	N            int `json:"n"`
	ScriptPubKey struct {
		Addresses []string `json:"addresses"`
		Asm       string   `json:"asm"`
		Hex       string   `json:"hex"`
		ReqSigs   int      `json:"reqSigs"`
		Type      string   `json:"type"`
	} `json:"scriptPubKey"`
	Value    float64 `json:"value"`
	ValueZat int64   `json:"valueZat"`
}

//----------------------------------------------------------------------
// Helper functions
//----------------------------------------------------------------------

func HTTPQuery(ctx context.Context, query string) ([]byte, error) {
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
	return io.ReadAll(resp.Body)
}
