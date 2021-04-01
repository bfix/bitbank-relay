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
	"os"
	"testing"
)

func TestMarketUpdate(t *testing.T) {
	apiKey := os.Getenv("COINAPI_APIKEY")
	symbols := "btc,bch,btg,dash,dgb,doge,ltc,nmc,vtc,zec,eth,etc"
	data, err := GetMarketData("EUR", symbols, apiKey)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range data {
		t.Logf("%s=%f\n", k, v)
	}
}
