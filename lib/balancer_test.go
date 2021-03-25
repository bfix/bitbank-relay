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
	"math"
	"testing"
)

// get the balance for an address and handle "acceptable" failures with
// a negative balance.
func _get(t *testing.T, b Balancer, addr string) (float64, bool) {
	val, err := b(addr)
	ok := true
	if err != nil {
		if err != ErrBalanceAccessDenied && err != ErrBalanceFailed {
			t.Fatal(err)
		}
		t.Log("Failed: " + err.Error())
		ok = false
	}
	return val, ok
}

func TestBalancerBtcEmpty(t *testing.T) {
	addr := "3EtzTLkZznFz9p7XVvVkEKSfqWtgFoPfeu"
	val, ok := _get(t, BtcBalancer, addr)
	if ok && math.Abs(val) >= 1e-8 {
		t.Fatal("Balance not ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceBtcUsed(t *testing.T) {
	addr := "1DFrkMZnFReDz93FQPBLT512DBDPAFF6qV"
	val, ok := _get(t, BtcBalancer, addr)
	if ok && math.Abs(val) < 1e-8 {
		t.Fatal("Balance ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceEthEmpty(t *testing.T) {
	addr := "0x2c95f5d417742747a9c3c9c4110191e4d684c9da"
	val, ok := _get(t, EthBalancer, addr)
	if ok && math.Abs(val) >= 1e-8 {
		t.Fatal("Balance not ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceEthUsed(t *testing.T) {
	addr := "0x3f5ce5fbfe3e9af3971dd833d26ba9b5c936f0be"
	val, ok := _get(t, EthBalancer, addr)
	if ok && math.Abs(val) < 1e-8 {
		t.Fatal("Balance is ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceZecEmpty(t *testing.T) {
	addr := "t1dZ5Tz8CqnhuQCjeUDrC7xMYtixpyykQ1b"
	val, ok := _get(t, ZecBalancer, addr)
	if ok && math.Abs(val) >= 1e-8 {
		t.Fatal("Balance not ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceZecUsed(t *testing.T) {
	addr := "t3XyYW8yBFRuMnfvm5KLGFbEVz25kckZXym"
	val, ok := _get(t, ZecBalancer, addr)
	if ok && math.Abs(val) < 1e-8 {
		t.Fatal("Balance is ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceBitcoinCashEmpty(t *testing.T) {
	addr := "qpnfc27ttwqky82emu6mvwtqphg94y4ahc957hjwhp"
	val, ok := _get(t, BchBalancer, addr)
	if ok && math.Abs(val) >= 1e-8 {
		t.Fatal("Balance not ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceBitcoinCashUsed(t *testing.T) {
	addr := "qz7xc0vl85nck65ffrsx5wvewjznp9lflgktxc5878"
	val, ok := _get(t, BchBalancer, addr)
	if ok && math.Abs(val) < 1e-8 {
		t.Fatal("Balance is ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceDogeEmpty(t *testing.T) {
	addr := "DTAfQ9aRZLue1bmFjcpnWadzoyiieGKHg5"
	val, ok := _get(t, DogeBalancer, addr)
	if ok && math.Abs(val) >= 1e-8 {
		t.Fatal("Balance not ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceDogeUsed(t *testing.T) {
	addr := "DH5yaieqoZN36fDVciNyRueRGvGLR3mr7L"
	val, ok := _get(t, DogeBalancer, addr)
	if ok && math.Abs(val) < 1e-8 {
		t.Fatal("Balance is ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceDashEmpty(t *testing.T) {
	addr := "XpXdvoyijeEingcVni5kPCTqXHL6as7Uxv"
	val, ok := _get(t, DashBalancer, addr)
	if ok && math.Abs(val) >= 1e-8 {
		t.Fatal("Balance not ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceDashUsed(t *testing.T) {
	addr := "XcQgFcjNLS36B6TYAW8ZrbibZC31Rbxitg"
	val, ok := _get(t, DashBalancer, addr)
	if ok && math.Abs(val) < 1e-8 {
		t.Fatal("Balance is ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceLitecoinEmpty(t *testing.T) {
	addr := "MNa6k9obZu1QezfF8mHxZ4fvFD5c126WfE"
	val, ok := _get(t, LtcBalancer, addr)
	if ok && math.Abs(val) >= 1e-8 {
		t.Fatal("Balance not ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceLitecoinUsed(t *testing.T) {
	addr := "M8T1B2Z97gVdvmfkQcAtYbEepune1tzGua"
	val, ok := _get(t, LtcBalancer, addr)
	if ok && math.Abs(val) < 1e-8 {
		t.Fatal("Balance is ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceBitcoinGoldEmpty(t *testing.T) {
	addr := "AMtFjuyrExnQb2Bq5mattQrKvH7rWZH6BJ"
	val, ok := _get(t, BtgBalancer, addr)
	if ok && math.Abs(val) >= 1e-8 {
		t.Fatal("Balance not ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceBitcoinGoldUsed(t *testing.T) {
	addr := "AZtTR1fK9UWfvTt1fKLoBDA1vUrApePSLi"
	val, ok := _get(t, BtgBalancer, addr)
	if ok && math.Abs(val) < 1e-8 {
		t.Fatal("Balance is ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceVertcoinEmpty(t *testing.T) {
	addr := "35kk2Zc56w52GH4koqEM5CQ73HFtvKZ9Am"
	val, ok := _get(t, VtcBalancer, addr)
	if ok && math.Abs(val) >= 1e-8 {
		t.Fatal("Balance not ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

func TestBalanceVertcoinUsed(t *testing.T) {
	addr := "VfukW89WKT9h3YjHZdSAAuGNVGELY31wyj"
	val, ok := _get(t, VtcBalancer, addr)
	if ok && math.Abs(val) < 1e-8 {
		t.Fatal("Balance is ZERO!")
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
}

// DGB min SfQvH8fzBJSSD4tYVN9ms8kHGVaHvQiqzd
