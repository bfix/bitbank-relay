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
	"strconv"
)

//======================================================================
// ETH (Ethereum)
//======================================================================

// EthChainHandler handles ETH-related blockchain operations
type EthChainHandler struct {
	BasicChainHandler
}

// Init a new chain handler instance
func (hdlr *EthChainHandler) Init(cfg *HandlerConfig) {
	hdlr.BasicChainHandler.Init(cfg)
	if hdlr.apiKey == "" {
		hdlr.apiKey = "freekey"
	}
}

// Balance gets the balance of an Ethereum address
func (hdlr *EthChainHandler) Balance(addr string) (float64, error) {
	// only handle one call at a time
	hdlr.lock.Lock()
	defer hdlr.lock.Unlock()

	// perform query
	hdlr.ratelimiter.Pass()
	query := fmt.Sprintf("https://api.ethplorer.io/getAddressInfo/%s?showETHTotals=true&apiKey=%s", addr, hdlr.apiKey)
	body, err := ChainQuery(context.Background(), query)
	if err != nil {
		return -1, err
	}
	data := new(EthAddrInfo)
	if err = json.Unmarshal(body, &data); err != nil {
		return -1, err
	}
	// return balance (incoming funds)
	return data.ETH.TotalIn, nil
}

// GetFunds returns incoming transaction for an Ethereum address.
func (hdlr *EthChainHandler) GetFunds(ctx context.Context, addrId int64, addr string) ([]*Fund, error) {
	// only handle one call at a time
	hdlr.lock.Lock()
	defer hdlr.lock.Unlock()

	// perform query
	hdlr.ratelimiter.Pass()
	query := fmt.Sprintf("https://api.ethplorer.io/getAddressTransactions/%s?apiKey=%s", addr, hdlr.apiKey)
	body, err := ChainQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	data := make([]*EthTxInfo, 0)
	if err = json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	// find received funds in transaction outputs
	funds := make([]*Fund, 0)
	for _, tx := range data {
		f := &Fund{
			Seen:   tx.Time,
			Addr:   addrId,
			Amount: tx.Value,
		}
		funds = append(funds, f)
	}
	// return funds
	return funds, nil
}

//----------------------------------------------------------------------
// internal access helpers
//----------------------------------------------------------------------

// EthAddrInfo is a response for an address info query
type EthAddrInfo struct {
	Address string `json:"address"`
	ETH     struct {
		Balance  float64 `json:"balance"`
		TotalIn  float64 `json:"totalIn"`
		TotalOut float64 `json:"totalOut"`
		Price    struct {
			Rate            float64 `json:"rate"`
			Diff            float64 `json:"diff"`
			Diff7d          float64 `json:"diff7d"`
			Ts              int64   `json:"ts"`
			MarketCapUSD    float64 `json:"marketCapUsd"`
			AvailableSupply float64 `json:"availableSupply"`
			Volume24h       float64 `json:"volume24h"`
			Diff30d         float64 `json:"diff30d"`
			VolDiff1        float64 `json:"volDiff1"`
			VolDiff7        float64 `json:"volDiff7"`
			VolDiff30       float64 `json:"volDiff30"`
		} `json:"price"`
	} `json:"ETH"`
	CounTxs int `json:"countTxs"`
}

// EthTxInfo is a response for an address transaction query
type EthTxInfo struct {
	Time    int64   `json:"timestamp"`
	From    string  `json:"from"`
	To      string  `json:"to"`
	Hash    string  `json:"hash"`
	Value   float64 `json:"value"`
	Input   string  `json:"input"`
	Success bool    `json:"success"`
}

//======================================================================
// ETC (Ethereum Classic)
//======================================================================

// EtcChainHandler handles Ethereum Classic-related blockchain operations
type EtcChainHandler struct {
	BasicChainHandler
}

// Balance gets the balance of an Ethereum address
func (hdlr *EtcChainHandler) Balance(addr string) (float64, error) {
	// only handle one call at a time
	hdlr.lock.Lock()
	defer hdlr.lock.Unlock()

	// perform query
	hdlr.ratelimiter.Pass()
	query := fmt.Sprintf("https://blockscout.com/etc/mainnet/api?module=account&action=balance&address=%s", addr)
	body, err := ChainQuery(context.Background(), query)
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
	return float64(val) / 1e8, nil
}

// GetFunds returns incoming transaction for an Ethereum address.
func (hdlr *EtcChainHandler) GetFunds(ctx context.Context, addrId int64, addr string) ([]*Fund, error) {
	// only handle one call at a time
	hdlr.lock.Lock()
	defer hdlr.lock.Unlock()

	// perform query
	hdlr.ratelimiter.Pass()
	query := fmt.Sprintf("https://blockscout.com/etc/mainnet/api?module=account&action=txlist&address=%s", addr)
	body, err := ChainQuery(ctx, query)
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
			Amount: float64(val) / 1e8,
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
