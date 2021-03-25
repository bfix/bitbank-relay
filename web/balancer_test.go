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
	b := new(BtcBalancer)
	val, err := b.Get(addr)
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
	b := new(BtcBalancer)
	val, err := b.Get(addr)
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
	b := new(EthBalancer)
	val, err := b.Get(addr)
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
	b := new(EthBalancer)
	val, err := b.Get(addr)
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
	b := new(ZecBalancer)
	val, err := b.Get(addr)
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
	b := new(ZecBalancer)
	val, err := b.Get(addr)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(val) < 1e-8 {
		t.Fatal("Balance is ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}
