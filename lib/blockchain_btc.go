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

	"github.com/bfix/gospel/network"
)

//======================================================================
// BTC (Bitcoin)
//======================================================================

// BtcChainHandler handles BTC-related blockchain operations
type BtcChainHandler struct {
	ratelimiter *network.RateLimiter
	limit       float64
	apiKey      string
	explorer    string
}

// Init a new chain handler instance
func (hdlr *BtcChainHandler) Init(cfg *HandlerConfig) {
	hdlr.ratelimiter = network.NewRateLimiter(cfg.Rates...)
	hdlr.limit = cfg.Limit
	hdlr.apiKey = cfg.ApiKey
	hdlr.explorer = cfg.Explorer
}

// Exporer returns the pattern for the blockchain browser URL
func (hdlr *BtcChainHandler) Explore(addr string) string {
	return hdlr.explorer
}

// Balance gets the balance of a Bitcoin address
func (hdlr *BtcChainHandler) Balance(addr string) (float64, error) {
	// perform query
	hdlr.ratelimiter.Pass()
	query := fmt.Sprintf("https://blockchain.info/rawaddr/%s", addr)
	body, err := ChainQuery(context.Background(), query)
	if err != nil {
		return -1, err
	}
	data := new(BtcAddrInfo)
	if err = json.Unmarshal(body, &data); err != nil {
		return -1, err
	}
	// return balance
	return float64(data.Received) / 1e8, nil
}

// GetFunds returns a list of incoming funds for the address
func (hdlr *BtcChainHandler) GetFunds(ctx context.Context, addrId int64, addr string) ([]*Fund, error) {
	// perform query
	hdlr.ratelimiter.Pass()
	query := fmt.Sprintf("https://blockchain.info/rawaddr/%s", addr)
	body, err := ChainQuery(context.Background(), query)
	if err != nil {
		return nil, err
	}
	data := new(BtcAddrInfo)
	if err = json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	// find received funds in transaction outputs
	funds := make([]*Fund, 0)
	for _, tx := range data.Transactions {
		for _, out := range tx.Outputs {
			if out.Addr == addr {
				f := &Fund{
					Seen:   tx.Time,
					Addr:   addrId,
					Amount: float64(out.Value / 1e8),
				}
				funds = append(funds, f)
			}
		}
	}
	// return funds
	return funds, nil
}

// Limit is the max. funding of an address (auto-close)
func (hdlr *BtcChainHandler) Limit() float64 {
	return hdlr.limit
}

//----------------------------------------------------------------------
// internal access helpers
//----------------------------------------------------------------------

type BtcAddrInfo struct {
	Hash160      string   `json:"hash160"`
	Address      string   `json:"address"`
	NTx          int      `json:"n_tx"`
	Nur          int      `json:"n_unredeemed"`
	Received     int64    `json:"total_received"`
	Sent         int64    `json:"total_sent"`
	Balance      int64    `json:"final_balance"`
	Transactions []*BtcTx `json:"txs"`
}

type BtcTx struct {
	Hash        string         `json:"hash"`
	Version     int            `json:"ver"`
	N_Vin       int            `json:"vin_sz"`
	N_Vout      int            `json:"vout_sz"`
	Size        int            `json:"size"`
	Weight      int            `json:"weight"`
	Fee         int64          `json:"fee"`
	Relay       string         `json:"relayed_by"`
	LockTime    int64          `json:"lock_time"`
	TxIndex     int64          `json:"tx_index"`
	DoubleSpend bool           `json:"double_spend"`
	Time        int64          `json:"time"`
	BlockIndex  int            `json:"block_index"`
	BlockHeight int            `json:"block_height"`
	Inputs      []*BtcTxInput  `json:"inputs"`
	Outputs     []*BtcTxOutput `json:"out"`
	Result      int64          `json:"result"`
	Balance     int64          `json:"balance"`
}

type BtcTxInput struct {
	Sequence int    `json:"sequence"`
	Witness  string `json:"witness"`
	Script   string `json:"script"`
	Index    int    `json:"index"`
	PrevOut  struct {
		Spent     bool           `json:"spent"`
		Script    string         `json:"script"`
		Spendings []*BtcSpending `json:"spending_outpoints"`
		TxIndex   int64          `json:"tx_index"`
		Value     int64          `json:"value"`
		Addr      string         `json:"addr"`
		N         int            `json:"n"`
		Type      int            `json:"type"`
	} `json:"prev_out"`
}

type BtcTxOutput struct {
	Type      int            `json:"type"`
	Spent     bool           `json:"spent"`
	Value     int64          `json:"value"`
	Spendings []*BtcSpending `json:"spending_outpoints"`
	N         int            `jsnon:"n"`
	TxIndex   int64          `json:"tx_index"`
	Script    string         `json:"script"`
	Addr      string         `json:"addr"`
}

type BtcSpending struct {
	TxIndex int64 `json:"tx_index"`
	N       int   `json:"n"`
}
