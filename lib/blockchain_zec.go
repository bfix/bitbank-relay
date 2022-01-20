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
// ZEC (ZCash)
//======================================================================

/// ZecChainHandler handles ZCash-related blockchain operations
type ZecChainHandler struct {
	BasicChainHandler
}

// Balance gets the balance of a ZCash address
func (hdlr *ZecChainHandler) Balance(addr string) (float64, error) {
	// only handle one call at a time
	hdlr.lock.Lock()
	defer hdlr.lock.Unlock()

	// assemble query
	hdlr.ratelimiter.Pass()
	query := fmt.Sprintf("https://api.zcha.in/v2/mainnet/accounts/%s", addr)
	body, err := ChainQuery(context.Background(), query)
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
func (hdlr *ZecChainHandler) GetFunds(ctx context.Context, addrId int64, addr string) ([]*Fund, error) {
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
		body, err := ChainQuery(ctx, query)
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
