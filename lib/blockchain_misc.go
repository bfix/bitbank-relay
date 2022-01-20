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

//======================================================================
// Delegated handlers (use a shared handler)
//======================================================================

//----------------------------------------------------------------------
// BCH (Bitcoin Cash)
//----------------------------------------------------------------------

// BchChainHandler handles Bitcoin Cash-related blockchain operations
type BchChainHandler struct {
	DerivedChainHandler
}

// Init chain handler
func (hdlr *BchChainHandler) Init(cfg *HandlerConfig) {
	hdlr.parent = bcHandler
	hdlr.coin = "bitcoin-cash"
	hdlr.parent.Init(cfg)
}

//----------------------------------------------------------------------
// DASH
//----------------------------------------------------------------------

// DashChainHandler handles Dash-related blockchain operations
type DashChainHandler struct {
	DerivedChainHandler
}

// Init chain handler
func (hdlr *DashChainHandler) Init(cfg *HandlerConfig) {
	hdlr.parent = bcHandler
	hdlr.coin = "dash"
	hdlr.parent.Init(cfg)
}

//----------------------------------------------------------------------
// DGB (Digibyte)
//----------------------------------------------------------------------

// DgbChainHandler handles Digibyte-related blockchain operations
type DgbChainHandler struct {
	DerivedChainHandler
}

// Init chain handler
func (hdlr *DgbChainHandler) Init(cfg *HandlerConfig) {
	hdlr.parent = cciHandler
	hdlr.coin = "dgb"
	hdlr.parent.Init(cfg)
}

//----------------------------------------------------------------------
// Doge (Dogecoin)
//----------------------------------------------------------------------

// DogeChainHandler handles Doge-related blockchain operations
type DogeChainHandler struct {
	DerivedChainHandler
}

// Init chain handler
func (hdlr *DogeChainHandler) Init(cfg *HandlerConfig) {
	hdlr.parent = bcHandler
	hdlr.coin = "dogecoin"
	hdlr.parent.Init(cfg)
}

//----------------------------------------------------------------------
// LTC (Litecoin)
//----------------------------------------------------------------------

// LtcChainHandler handles Litecoin-related blockchain operations
type LtcChainHandler struct {
	DerivedChainHandler
}

// Init chain handler
func (hdlr *LtcChainHandler) Init(cfg *HandlerConfig) {
	hdlr.parent = bcHandler
	hdlr.coin = "litecoin"
	hdlr.parent.Init(cfg)
}

//----------------------------------------------------------------------
// NMC (Namecoin)
//----------------------------------------------------------------------

// NmcChainHandler handles Namecoin-related blockchain operations
type NmcChainHandler struct {
	DerivedChainHandler
}

// Init chain handler
func (hdlr *NmcChainHandler) Init(cfg *HandlerConfig) {
	hdlr.parent = cciHandler
	hdlr.coin = "nmc"
	hdlr.parent.Init(cfg)
}

//----------------------------------------------------------------------
// VTC (Vertcoin)
//----------------------------------------------------------------------

// VtcChainHandler handles Vertcoin-related blockchain operations
type VtcChainHandler struct {
	DerivedChainHandler
}

// Init chain handler
func (hdlr *VtcChainHandler) Init(cfg *HandlerConfig) {
	hdlr.parent = cciHandler
	hdlr.coin = "vtc"
	hdlr.parent.Init(cfg)
}
