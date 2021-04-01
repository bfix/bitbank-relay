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
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// MarketDataResponse is a response from the Market API
type MarketDataResponse struct {
	Base  string `json:"asset_id_base"`
	Rates []*struct {
		Time string  `json:"time"`
		Coin string  `json:"asset_id_quote"`
		Rate float64 `json:"rate"`
	} `json:"rates"`
}

// GetMarketData returns the current rates for given currencies
func GetMarketData(fiat, symbols, apiKey string) (map[string]float64, error) {
	// assemble query
	query := fmt.Sprintf("https://rest.coinapi.io/v1/exchangerate/%s", fiat)
	client := &http.Client{}
	req, err := http.NewRequest("GET", query, nil)
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Add("filter_asset_id", symbols)
	req.Header.Set("Accepts", "application/json")
	req.Header.Add("X-CoinAPI-Key", apiKey)
	req.URL.RawQuery = q.Encode()

	// send query and receive response
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// parse response
	data := new(MarketDataResponse)
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	// assemble result
	res := make(map[string]float64)
	for _, rate := range data.Rates {
		res[strings.ToLower(rate.Coin)] = 1. / rate.Rate
	}
	return res, nil
}
