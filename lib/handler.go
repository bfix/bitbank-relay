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
	"fmt"
	"strings"

	"github.com/bfix/gospel/bitcoin"
	"github.com/bfix/gospel/bitcoin/script"
	"github.com/bfix/gospel/bitcoin/wallet"
)

var (
	// HdlrList is a list of registered handlers
	HdlrList = make(map[string]*Handler)
)

// Handler to handle coin accounts (in BIP44/49 wallets)
type Handler struct {
	coin     int              // coin identifier (BIP-32)
	symb     string           // coin symbol
	mode     int              // address mode (P2PKH, P2SH, ...)
	netw     int              // network (Main, Test, Reg)
	tree     *wallet.HDPublic // HDKD for public keys
	pathTpl  string           // path template for indexing addresses
	limit    float64          // auto-close balance on address
	explorer string           // Explorer URL for address
	chain    ChainHandler     // blockchain handler for coin
	market   MarketHandler    // market handler for coin
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

	// get coin identifier and handlers
	coinID, _ := wallet.GetCoinInfo(coin.Symb)
	chainHdlr, ok := baseChainHdlrs[coin.Blockchain]
	if !ok {
		return nil, fmt.Errorf("no blockchain handler for coin %s", coin.Symb)
	}
	var marketHdlr MarketHandler = nil

	// assemble handler for given coin
	return &Handler{
		coin:     coinID,
		symb:     coin.Symb,
		mode:     coin.GetMode(),
		netw:     network,
		tree:     wallet.NewHDPublic(pk, coin.Path),
		pathTpl:  path,
		limit:    coin.Limit,
		explorer: coin.Explorer,
		chain:    chainHdlr,
		market:   marketHdlr,
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

	switch hdlr.mode {
	case wallet.AddrP2PKH, wallet.AddrP2WPKH, wallet.AddrP2WPKHinP2SH, -1:
		return wallet.MakeAddress(pk, hdlr.coin, hdlr.mode, hdlr.netw)
	case wallet.AddrP2SH:
		scr := script.NewScript()
		scr.Add(script.NewStatement(0))
		scr.Add(script.NewDataStatement(pk.Bytes()))
		return wallet.MakeAddressScript(scr, hdlr.coin, hdlr.mode, hdlr.netw)
	default:
		return "", wallet.ErrMkAddrVersion
	}
}

// GetBalance returns the balance for a given address
func (hdlr *Handler) GetBalance(ctx context.Context, addr string) (float64, error) {
	// call balance function
	return hdlr.chain.Balance(ctx, addr, hdlr.symb)
}

// GetTxList returns a list of transaction for an address
func (hdlr *Handler) GetFunds(ctx context.Context, addrId int64, addr string) ([]*Fund, error) {
	// call reporting function
	return hdlr.chain.GetFunds(ctx, addrId, addr, hdlr.symb)
}

//----------------------------------------------------------------------
// Setup handler list from configuration

func InitHandlers(cfg *Config, mdl *Model) (coins []string, err error) {

	// initialize shared handler instances:
	// ------------------------------------
	// (1) blockchain handlers
	for name, hdlrCfg := range cfg.Handler.Blockchain {
		if hdlr, ok := baseChainHdlrs[name]; ok {
			hdlr.Init(hdlrCfg)
		}
	}
	// (2) market handlers
	for name, hdlrCfg := range cfg.Handler.Market.Service {
		if hdlr, ok := baseMarketHdlrs[name]; ok {
			hdlr.Init(hdlrCfg)
		}
	}

	// load actual coin handlers; assemble list of coin symbols
	for _, coin := range cfg.Coins {
		// check if coin is in model
		if _, err = mdl.GetCoin(coin.Symb); err != nil {
			return
		}
		// add to list of coins
		coins = append(coins, coin.Symb)
		// get coin handler
		var hdlr *Handler
		if hdlr, err = NewHandler(coin, wallet.NetwMain); err != nil {
			return
		}
		// verify handler
		var addr string
		if addr, err = hdlr.GetAddress(0); err != nil {
			return
		}
		if addr != coin.Addr {
			err = fmt.Errorf("addr mismatch: %s != %s", addr, coin.Addr)
			return
		}
		// save handler
		HdlrList[coin.Symb] = hdlr
	}
	return
}

//----------------------------------------------------------------------
// helper functions

// GetNetwork returns the numeric coin network ID
func GetNetwork(netw string) int {
	switch strings.ToLower(netw) {
	case "main":
		return wallet.NetwMain
	case "test":
		return wallet.NetwTest
	case "reg":
		return wallet.NetwReg
	}
	return -1
}
