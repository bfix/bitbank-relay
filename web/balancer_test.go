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

package main

import (
	"math"
	"testing"
)

func TestBalancerBtcEmpty(t *testing.T) {
	addr := "3EtzTLkZznFz9p7XVvVkEKSfqWtgFoPfeu"
	val, err := BtcBalancer(addr)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(val) >= 1e-8 {
		t.Fatal("Balance not ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceBtcUsed(t *testing.T) {
	addr := "1DFrkMZnFReDz93FQPBLT512DBDPAFF6qV"
	val, err := BtcBalancer(addr)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(val) < 1e-8 {
		t.Fatal("Balance ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceEthEmpty(t *testing.T) {
	addr := "0x2c95f5d417742747a9c3c9c4110191e4d684c9da"
	val, err := EthBalancer(addr)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(val) >= 1e-8 {
		t.Fatal("Balance not ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceEthUsed(t *testing.T) {
	addr := "0x3f5ce5fbfe3e9af3971dd833d26ba9b5c936f0be"
	val, err := EthBalancer(addr)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(val) < 1e-8 {
		t.Fatal("Balance is ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceZecEmpty(t *testing.T) {
	addr := "t1dZ5Tz8CqnhuQCjeUDrC7xMYtixpyykQ1b"
	val, err := ZecBalancer(addr)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(val) >= 1e-8 {
		t.Fatal("Balance not ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceZecUsed(t *testing.T) {
	addr := "t3XyYW8yBFRuMnfvm5KLGFbEVz25kckZXym"
	val, err := ZecBalancer(addr)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(val) < 1e-8 {
		t.Fatal("Balance is ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceBitcoinCashEmpty(t *testing.T) {
	addr := "qpnfc27ttwqky82emu6mvwtqphg94y4ahc957hjwhp"
	val, err := BchBalancer(addr)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(val) >= 1e-8 {
		t.Fatal("Balance not ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceBitcoinCashUsed(t *testing.T) {
	addr := "qz7xc0vl85nck65ffrsx5wvewjznp9lflgktxc5878"
	val, err := BchBalancer(addr)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(val) < 1e-8 {
		t.Fatal("Balance is ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceDogeEmpty(t *testing.T) {
	addr := "DTAfQ9aRZLue1bmFjcpnWadzoyiieGKHg5"
	val, err := DogeBalancer(addr)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(val) >= 1e-8 {
		t.Fatal("Balance not ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceDogeUsed(t *testing.T) {
	addr := "DH5yaieqoZN36fDVciNyRueRGvGLR3mr7L"
	val, err := DogeBalancer(addr)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(val) < 1e-8 {
		t.Fatal("Balance is ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceDashEmpty(t *testing.T) {
	addr := "XpXdvoyijeEingcVni5kPCTqXHL6as7Uxv"
	val, err := DashBalancer(addr)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(val) >= 1e-8 {
		t.Fatal("Balance not ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceDashUsed(t *testing.T) {
	addr := "XcQgFcjNLS36B6TYAW8ZrbibZC31Rbxitg"
	val, err := DashBalancer(addr)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(val) < 1e-8 {
		t.Fatal("Balance is ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceLitecoinEmpty(t *testing.T) {
	addr := "MNa6k9obZu1QezfF8mHxZ4fvFD5c126WfE"
	val, err := LtcBalancer(addr)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(val) >= 1e-8 {
		t.Fatal("Balance not ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceLitecoinUsed(t *testing.T) {
	addr := "M8T1B2Z97gVdvmfkQcAtYbEepune1tzGua"
	val, err := LtcBalancer(addr)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(val) < 1e-8 {
		t.Fatal("Balance is ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}
