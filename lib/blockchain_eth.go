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
	// perform query
	hdlr.ratelimiter.Pass()
	query := fmt.Sprintf("https://api.ethplorer.io/getAddressTransactions/%s?apiKey=%s", addr, hdlr.apiKey)
	body, err := ChainQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	data := make([]*EthAddrTx, 0)
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

// EthAddrTx is a response for an address transaction query
type EthAddrTx struct {
	Time    int64   `json:"timestamp"`
	From    string  `json:"from"`
	To      string  `json:"to"`
	Hash    string  `json:"hash"`
	Value   float64 `json:"value"`
	Input   string  `json:"input"`
	Success bool    `json:"success"`
}
