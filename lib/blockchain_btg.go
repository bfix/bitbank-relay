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
// BTG (Bitcoin Gold)
//======================================================================

// BtgChainHandler handles BitcoinGold-related blockchain operations
type BtgChainHandler struct {
	BasicChainHandler
}

// Balance gets the balance of a Bitcoin Gold address
func (hdlr *BtgChainHandler) Balance(addr string) (float64, error) {
	// only handle one call at a time
	hdlr.lock.Lock()
	defer hdlr.lock.Unlock()

	// perform query
	hdlr.ratelimiter.Pass()
	query := fmt.Sprintf("https://btgexplorer.com/api/address/%s", addr)
	body, err := ChainQuery(context.Background(), query)
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
func (hdlr *BtgChainHandler) GetFunds(ctx context.Context, addrId int64, addr string) ([]*Fund, error) {
	// only handle one call at a time
	hdlr.lock.Lock()
	defer hdlr.lock.Unlock()

	// perform query (stage 1)
	hdlr.ratelimiter.Pass()
	query := fmt.Sprintf("https://btgexplorer.com/api/address/%s", addr)
	body, err := ChainQuery(context.Background(), query)
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
		body, err := ChainQuery(ctx, query)
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
