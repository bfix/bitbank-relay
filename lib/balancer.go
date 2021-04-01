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

	"github.com/bfix/gospel/logger"
)

// Balancer prototype for querying address balances
type Balancer func(addr string) (float64, error)

//----------------------------------------------------------------------
// Manage available balancers
//----------------------------------------------------------------------

// List of known address balancers
var (
	balancer = map[string]Balancer{
		"btc":  BtcBalancer,
		"bch":  BchBalancer,
		"btg":  BtgBalancer,
		"dash": DashBalancer,
		"dgb":  DgbBalancer,
		"doge": DogeBalancer,
		"ltc":  LtcBalancer,
		"nmc":  nil,
		"vtc":  VtcBalancer,
		"zec":  ZecBalancer,
		"eth":  EthBalancer,
		"etc":  nil,
	}
)

// Error codes
var (
	ErrBalanceFailed       = fmt.Errorf("balance query failed")
	ErrBalanceAccessDenied = fmt.Errorf("HTTP GET access denied")
)

// StartBalancer starts the background balance processor.
// It returns a channel for balance check requests that accepts int64
// values that refer to the database id (row id) of the address record
// that is to be checked.
func StartBalancer(ctx context.Context, db *Database, cfg *BalancerConfig) chan int64 {
	ch := make(chan int64)
	go func() {
		for {
			select {
			// handle balance requests
			case ID := <-ch:
				logger.Printf(logger.DBG, "Balancer: request=%d", ID)

				// close processor on negative row id
				if ID < 0 {
					return
				}
				// get address information
				var (
					addr, coin string
					balance    float64
				)
				row := db.inst.QueryRow("select coin,val,balance from v_addr where id=?", ID)
				if err := row.Scan(&coin, &addr, &balance); err != nil {
					logger.Printf(logger.ERROR, "Balancer: can't retrieve address #%d", ID)
					logger.Println(logger.ERROR, "=> "+err.Error())
					continue
				}
				// get new address balance
				hdlr, ok := HdlrList[coin]
				if !ok {
					logger.Printf(logger.ERROR, "Balancer: No handler for '%s'", coin)
					continue
				}
				newBalance, err := hdlr.GetBalance(addr)
				if err != nil {
					logger.Println(logger.ERROR, "Balancer: "+err.Error())
					continue
				}
				// update balance if increased
				if newBalance >= balance {
					balance = newBalance
				}
				// update balance
				if _, err = db.inst.Exec(
					"update addr set balance=?, last_check=? where id=?",
					balance, time.Now().Unix(), ID); err != nil {
					logger.Println(logger.ERROR, "Balance update: "+err.Error())
				}
				// check if limit is reached...
				if hdlr.CheckClose(cfg.AccountLimit, newBalance) {
					// yes: close address
					logger.Printf(logger.INFO, "Closing address '%s' with balance=%f", addr, newBalance)
					_, err := db.inst.Exec("update addr set stat=1, validTo=now() where id=?", ID)
					if err != nil {
						logger.Println(logger.ERROR, "Balancer: "+err.Error())
					}
				}

			// cancel processor
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}

//----------------------------------------------------------------------
// BTC (Bitcoin)
//----------------------------------------------------------------------

// BtcAddrInfo is a response from the blockchain.info API when
// querying BTC address balances.
type BtcAddrInfo struct {
	Hash160       string `json:"hash160"`
	Address       string `json:"address"`
	NumTx         int    `json:"n_tx"`
	NumUtxo       int    `json:"n_unredeemed"`
	TotalReceived int64  `json:"total_received"`
	TotalSend     int64  `json:"total_sent"`
	FinalBalance  int64  `json:"final_balance"`
	Txs           []struct {
		ID          string `json:"hash"`
		Version     int    `json:"ver"`
		NumVin      int    `json:"vin_sz"`
		NumVout     int    `json:"vout_sz"`
		LockTime    int    `json:"lock_time"`
		Size        int    `json:"size"`
		RelayedBy   string `json:"relayed_by"`
		BlockHeight int    `json:"block_height"`
		TxIndex     int    `json:"tx_index"`
		Inputs      []struct {
			PrevOut struct {
				ID      string `json:"hash"`
				Value   int64  `json:"value"`
				TxIndex int    `json:"tx_index"`
				N       int    `json:"n"`
			} `json:"prev_out"`
			Script string `json:"script"`
		} `json:"inputs"`
		Outputs []struct {
			ID     string `json:"hash"`
			Value  int64  `json:"value"`
			Script string `json:"script"`
		} `json:"out"`
	} `json:"txs"`
}

// BtcBalancer gets the balance of a Bitcoin address
func BtcBalancer(addr string) (float64, error) {
	query := fmt.Sprintf("https://blockchain.info/rawaddr/%s?limit=0", addr)
	resp, err := http.Get(query)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}
	data := new(BtcAddrInfo)
	if err = json.Unmarshal(body, &data); err != nil {
		return -1, err
	}
	return float64(data.TotalReceived) / 1e8, nil
}

//----------------------------------------------------------------------
// ETH (Ethereum)
//----------------------------------------------------------------------

// EthAddressInfo is a response from the ethplorer.io API for an address query
type EthAddrInfo struct {
	Address string `json:"address"`
	ETH     struct {
		Balance float64 `json:"balance"`
		Price   struct {
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

// honor rate limit minimum
var ethLimiter = NewLimiter(5, 50, 200, 2000, 3000)

// EthBalancer gets the balance of an Ethereum address
func EthBalancer(addr string) (float64, error) {
	// honor rate limit
	ethLimiter.Pass()

	// assemble and execute GET request
	query := fmt.Sprintf("https://api.ethplorer.io/getAddressInfo/%s?apiKey=freekey", addr)
	resp, err := http.Get(query)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}
	data := new(EthAddrInfo)
	if err = json.Unmarshal(body, &data); err != nil {
		return -1, err
	}
	return data.ETH.Balance, nil
}

//----------------------------------------------------------------------
// ZEC (ZCash)
//----------------------------------------------------------------------

// ZecAddressInfo is a response from the zcha.in API for an address query
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

// ZecBalancer gets the balance of a ZCash address
func ZecBalancer(addr string) (float64, error) {
	query := fmt.Sprintf("https://api.zcha.in/v2/mainnet/accounts/%s", addr)
	resp, err := http.Get(query)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}
	data := new(ZecAddrInfo)
	if err = json.Unmarshal(body, &data); err != nil {
		return -1, err
	}
	return data.Balance, nil
}

//----------------------------------------------------------------------
// BCH (Bitcoin Cash)
//----------------------------------------------------------------------

// BchBalancer gets the balance of a Bitcoin Cash address
func BchBalancer(addr string) (float64, error) {
	data, err := BlockchairGet("bitcoin-cash", addr)
	if err != nil {
		return -1, err
	}
	return float64(data.Data[addr].Address.Balance) / 1e8, nil
}

//----------------------------------------------------------------------
// BTG (Bitcoin Gold)
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

// BtgBalancer gets the balance of a Bitcoin Gold address
func BtgBalancer(addr string) (float64, error) {
	query := fmt.Sprintf("https://btgexplorer.com/api/address/%s", addr)
	resp, err := http.Get(query)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}
	data := new(BtgAddrInfo)
	if err = json.Unmarshal(body, &data); err != nil {
		return -1, err
	}
	val, err := strconv.ParseFloat(data.Balance, 64)
	if err != nil {
		return -1, err
	}
	return val, nil
}

//----------------------------------------------------------------------
// DASH
//----------------------------------------------------------------------

// DashBalancer gets the balance of a Dash address
func DashBalancer(addr string) (float64, error) {
	data, err := BlockchairGet("dash", addr)
	if err != nil {
		return -1, err
	}
	return float64(data.Data[addr].Address.Balance) / 1e8, nil
}

//----------------------------------------------------------------------
// Doge (Dogecoin)
//----------------------------------------------------------------------

// DogeBalancer gets the balance of a Dogecoin address
func DogeBalancer(addr string) (float64, error) {
	data, err := BlockchairGet("dogecoin", addr)
	if err != nil {
		return -1, err
	}
	return float64(data.Data[addr].Address.Balance) / 1e8, nil
}

//----------------------------------------------------------------------
// LTC (Litecoin)
//----------------------------------------------------------------------

// LtcBalancer gets the balance of a Litecoin address
func LtcBalancer(addr string) (float64, error) {
	data, err := BlockchairGet("litecoin", addr)
	if err != nil {
		return -1, err
	}
	return float64(data.Data[addr].Address.Balance) / 1e8, nil
}

//----------------------------------------------------------------------
// VTC (Vertcoin)
//----------------------------------------------------------------------

var vtcLimiter = NewLimiter(0, 6)

// VtcBalancer gets the balance of a Namecoin address
func VtcBalancer(addr string) (float64, error) {
	// honor rate limit
	vtcLimiter.Pass()

	// assemble query
	query := fmt.Sprintf("https://chainz.cryptoid.info/vtc/api.dws?q=getbalance&a=%s", addr)
	resp, err := http.Get(query)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}
	val, err := strconv.ParseFloat(string(body), 64)
	if err != nil {
		return -1, err
	}
	return val, nil
}

//----------------------------------------------------------------------
// DGB (Digibyte)
//----------------------------------------------------------------------

var dgbLimiter = NewLimiter(0, 6)

// DgbBalancer gets the balance of a Namecoin address
func DgbBalancer(addr string) (float64, error) {
	// honor rate limit
	dgbLimiter.Pass()

	// assemble query
	query := fmt.Sprintf("https://chainz.cryptoid.info/dgb/api.dws?q=getbalance&a=%s", addr)
	resp, err := http.Get(query)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}
	val, err := strconv.ParseFloat(string(body), 64)
	if err != nil {
		return -1, err
	}
	return val, nil
}

//----------------------------------------------------------------------
// Generic Balancer (blockchair.com)
//----------------------------------------------------------------------

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

var bchairLimiter = NewLimiter(0, 1)

// BlockchairGet gets the balance of a Blockchair address
func BlockchairGet(coin, addr string) (*BlockchairAddrInfo, error) {
	bchairLimiter.Pass()

	// query API
	query := fmt.Sprintf("https://api.blockchair.com/%s/dashboards/address/%s", coin, addr)
	resp, err := http.Get(query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// read response
	body, err := ioutil.ReadAll(resp.Body)
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
	// return response
	return data, nil
}
