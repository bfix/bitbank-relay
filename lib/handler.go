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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bfix/gospel/bitcoin"
	"github.com/bfix/gospel/bitcoin/wallet"
)

var (
	// HdlrList is a list of registered handlers
	HdlrList = make(map[string]*Handler)
)

// Handler to handle coin accounts (in BIP44/49 wallets)
type Handler struct {
	coin     int              // coin identifier
	symb     string           // coin symbol
	mode     int              // address mode (P2PKH, P2SH, ...)
	netw     int              // network (Main, Test, Reg)
	tree     *wallet.HDPublic // HDKD for public keys
	balancer Balancer         // address balance handler for coin
	pathTpl  string           // path template for indexing addresses
	explorer string           // Explorer URL for address
}

// NewHandler creates a new handler instance for the given coin on
// a network (main/test/reg) if applicable
func NewHandler(coin *CoinConfig, network int) (*Handler, error) {

	// compute base account address
	pk, err := wallet.ParseExtendedPublicKey(coin.Pk)
	if err != nil {
		return nil, err
	}
	pk.Data.Version = coin.GetXDVersion()

	// compute path template for indexed addreses
	path := coin.Path
	for strings.Count(path, "/") < 4 {
		path += "/0"
	}
	path += "/%d"

	// get coin identifier
	coinID, _ := wallet.GetCoinInfo(coin.Symb)

	// balancer function for coin
	b, ok := balancer[coin.Symb]
	if !ok {
		b = nil
	}
	// assemble handler for given coin
	return &Handler{
		coin:     coinID,
		symb:     coin.Symb,
		mode:     coin.GetMode(),
		netw:     network,
		tree:     wallet.NewHDPublic(pk, coin.Path),
		balancer: b,
		pathTpl:  path,
		explorer: coin.Explorer,
	}, nil
}

// GetAddress returns the address for a given index in the account
func (hdlr *Handler) GetAddress(idx int) (string, error) {

	// get extended public key for indexed address
	epk, err := hdlr.tree.Public(fmt.Sprintf(hdlr.pathTpl, idx))
	if err != nil {
		return "", err
	}
	ed := epk.Data

	// get public key data
	pk, err := bitcoin.PublicKeyFromBytes(ed.Keydata)
	if err != nil {
		return "", err
	}
	return wallet.MakeAddress(pk, hdlr.coin, hdlr.mode, hdlr.netw), nil
}

// GetBalance returns the balance for a given address
func (hdlr *Handler) GetBalance(addr string) (float64, error) {
	return hdlr.balancer(addr)
}

//----------------------------------------------------------------------
// helper functions

// GetNetwork returns the numeric coin network ID
func GetNetwork(netw string) int {
	switch strings.ToLower(netw) {
	case "main":
		return wallet.AddrMain
	case "test":
		return wallet.AddrTest
	case "reg":
		return wallet.AddrReg
	}
	return -1
}

//----------------------------------------------------------------------
// shared blockchain APIs
//----------------------------------------------------------------------

// Blockcypher works for: BTC, LTC, DASH, DOGE, ETH
// Checks if an address is used (#tx > 0)
func Blockcypher(coin, addr string) (bool, error) {
	query := fmt.Sprintf("https://api.blockcypher.com/v1/%s/main/addrs/%s", coin, addr)
	resp, err := http.Get(query)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	var data map[string]interface{}
	if err = json.Unmarshal(body, &data); err != nil {
		return false, err
	}
	val, ok := data["n_tx"]
	if !ok {
		return false, fmt.Errorf("no 'n_tx' attribute")
	}
	n, ok := val.(uint64)
	if !ok {
		return false, fmt.Errorf("invalid 'n_tx' type")
	}
	return n > 0, nil
}
