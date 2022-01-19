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
	GenericChainHandler
}

// Balance gets the balance of a Bitcoin address
func (hdlr *BchChainHandler) Balance(addr string) (float64, error) {
	return bcHandler.Balance(addr, "bitcoin-cash")
}

//----------------------------------------------------------------------
// DASH
//----------------------------------------------------------------------

// DashChainHandler handles Dash-related blockchain operations
type DashChainHandler struct {
	GenericChainHandler
}

// Balance gets the balance of a Bitcoin address
func (hdlr *DashChainHandler) Balance(addr string) (float64, error) {
	return cciHandler.Balance(addr, "dash")
}

//----------------------------------------------------------------------
// Doge (Dogecoin)
//----------------------------------------------------------------------

// DogeChainHandler handles Doge-related blockchain operations
type DogeChainHandler struct {
	GenericChainHandler
}

// Balance gets the balance of a Bitcoin address
func (hdlr *DogeChainHandler) Balance(addr string) (float64, error) {
	return bcHandler.Balance(addr, "dogecoin")
}

//----------------------------------------------------------------------
// LTC (Litecoin)
//----------------------------------------------------------------------

// LtcChainHandler handles Litecoin-related blockchain operations
type LtcChainHandler struct {
	GenericChainHandler
}

// Balance gets the balance of a Bitcoin address
func (hdlr *LtcChainHandler) Balance(addr string) (float64, error) {
	return cciHandler.Balance(addr, "ltc")
}

//----------------------------------------------------------------------
// VTC (Vertcoin)
//----------------------------------------------------------------------

// VtcChainHandler handles Vertcoin-related blockchain operations
type VtcChainHandler struct {
	GenericChainHandler
}

// Balance gets the balance of a Bitcoin address
func (hdlr *VtcChainHandler) Balance(addr string) (float64, error) {
	return cciHandler.Balance(addr, "vtc")
}

//----------------------------------------------------------------------
// DGB (Digibyte)
//----------------------------------------------------------------------

// DgbChainHandler handles Digibyte-related blockchain operations
type DgbChainHandler struct {
	GenericChainHandler
}

// Balance gets the balance of a Bitcoin address
func (hdlr *DgbChainHandler) Balance(addr string) (float64, error) {
	return cciHandler.Balance(addr, "dgb")
}

//----------------------------------------------------------------------
// NMC (Namecoin)
//----------------------------------------------------------------------

// NmcChainHandler handles Namecoin-related blockchain operations
type NmcChainHandler struct {
	GenericChainHandler
}

// Balance gets the balance of a Namecoin address
func (hdlr *NmcChainHandler) Balance(addr string) (float64, error) {
	return 0, nil
}

//----------------------------------------------------------------------
// ETC (Ethereum Classic)
//----------------------------------------------------------------------

// EtcChainHandler handles Ethereum Classic-related blockchain operations
type EtcChainHandler struct {
	GenericChainHandler
}

// Balance gets the balance of a Namecoin address
func (hdlr *EtcChainHandler) Balance(addr string) (float64, error) {
	return 0, nil
}
