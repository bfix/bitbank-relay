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
	"os"
	"testing"
)

var (
	addrs = map[string][2]string{
		"btc":  {"3EtzTLkZznFz9p7XVvVkEKSfqWtgFoPfeu", "34xp4vRoCGJym3xR7yCVPFHoCNxv4Twseo"},
		"doge": {"DTAfQ9aRZLue1bmFjcpnWadzoyiieGKHg5", "DH5yaieqoZN36fDVciNyRueRGvGLR3mr7L"},
		"dash": {"XpXdvoyijeEingcVni5kPCTqXHL6as7Uxv", "XcQgFcjNLS36B6TYAW8ZrbibZC31Rbxitg"},
		"ltc":  {"MNa6k9obZu1QezfF8mHxZ4fvFD5c126WfE", "M8T1B2Z97gVdvmfkQcAtYbEepune1tzGua"},
		"btg":  {"AMtFjuyrExnQb2Bq5mattQrKvH7rWZH6BJ", "AZtTR1fK9UWfvTt1fKLoBDA1vUrApePSLi"},
		"vtc":  {"35kk2Zc56w52GH4koqEM5CQ73HFtvKZ9Am", "VfukW89WKT9h3YjHZdSAAuGNVGELY31wyj"},
		"dgb":  {"SfQvH8fzBJSSD4tYVN9ms8kHGVaHvQiqzd", "DBF4pdn7CGZMkxFUvdMbfxyJ8cRqfwsurj"},
		"nmc":  {"NBQyqsWZNZ4u2i11mqyj8y3NikaUa6t8Gk", "NAxZHe6yUCADnGAeCs4xrkgEKHjSFVrK5m"},
		"zec":  {"t1dZ5Tz8CqnhuQCjeUDrC7xMYtixpyykQ1b", "t3XyYW8yBFRuMnfvm5KLGFbEVz25kckZXym"},
		"bch":  {"qpnfc27ttwqky82emu6mvwtqphg94y4ahc957hjwhp", "qz7xc0vl85nck65ffrsx5wvewjznp9lflgktxc5878"},
		"eth":  {"0x2c95f5d417742747a9c3c9c4110191e4d684c9da", "0x3f5ce5fbfe3e9af3971dd833d26ba9b5c936f0be"},
		"etc":  {"0x31c582939bc2fb65ed7b6509647243c4aeb24c9f", "0x78D5E220B4cc84f290Fae4148831b371a851a114"},
	}
)

// get the balance for an address and handle "acceptable" failures with
// a negative balance.
func getBalance(t *testing.T, b Balancer, addr string) (float64, bool) {
	val, err := b(addr)
	if err != nil {
		if err != ErrBalanceAccessDenied && err != ErrBalanceFailed {
			t.Fatal(err)
		}
		t.Log("Failed: " + err.Error())
	}
	t.Logf("Balance for '%s': %f\n", addr, val)
	return val, err == nil
}

func checkEmpty(t *testing.T, b Balancer, addr string) {
	val, ok := getBalance(t, b, addr)
	if ok && math.Abs(val) >= 1e-8 {
		t.Fatal("Balance not ZERO!")
	}
}

func checkUsed(t *testing.T, b Balancer, addr string) {
	val, ok := getBalance(t, b, addr)
	if ok && math.Abs(val) < 1e-8 {
		t.Fatal("Balance is ZERO!")
	}
}

func TestBalances(t *testing.T) {
	// initialize API keys
	k := os.Getenv("BLOCKCHAIR_APIKEY")
	if len(k) > 0 {
		apikeys["blockchair"] = k
	}

	// test all addresses
	for coin, addr := range addrs {
		b := balancer[coin]
		if b == nil {
			continue
		}
		t.Log("--- " + coin)
		checkEmpty(t, b, addr[0])
		checkUsed(t, b, addr[1])
	}
}
