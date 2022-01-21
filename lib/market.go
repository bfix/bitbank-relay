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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// GetMarketData returns the current rates for given currencies.
func GetMarketData(ctx context.Context, fiat string, date int64, coins []string) (map[string]float64, error) {
	// we only have one handler at the moment...
	hdlr, ok := baseMarketHdlrs["coinapi.io"]
	if !ok {
		return nil, fmt.Errorf("no market handler found")
	}
	// handle one coin at a time
	list := make(map[string]float64)
	for _, coin := range coins {
		rate, err := hdlr.ExchangeRate(ctx, fiat, date, coin)
		if err != nil {
			return nil, err
		}
		list[coin] = rate
	}
	return list, nil
}

//----------------------------------------------------------------------
// Market handlers
//----------------------------------------------------------------------

// MarketHandler retrieves (historical) exchange rates for coins
type MarketHandler interface {
	Init(cfg *MarketHandlerConfig)
	ExchangeRate(ctx context.Context, fiat string, date int64, coin string) (float64, error)
}

var (
	// map of base market handlers
	baseMarketHdlrs = map[string]MarketHandler{
		"coinapi.io": new(CoinapiMarketHandler),
	}
)

// CoinapiMarketHandler handles exchange rate requests
type CoinapiMarketHandler struct {
	credits int64      // number of credits available
	apiKey  string     // API key for access
	lock    sync.Mutex // serializer
}

// Init handler from configuration
func (hdlr *CoinapiMarketHandler) Init(cfg *MarketHandlerConfig) {
	hdlr.apiKey = cfg.ApiKey
	hdlr.credits = 10
}

// ExchangeRate returns the exchange rates for a given date and list of coins.
func (hdlr *CoinapiMarketHandler) ExchangeRate(
	ctx context.Context,
	fiat string,
	date int64,
	coin string) (float64, error) {

	// serialize requests
	hdlr.lock.Lock()
	defer hdlr.lock.Unlock()
	// assemble query
	query := fmt.Sprintf("https://rest.coinapi.io/v1/exchangerate/%s/%s", strings.ToUpper(coin), fiat)
	if date > 0 {
		query += fmt.Sprintf("?time=%s", time.Unix(date, 0).Format("2006-01-02T15:04:05Z"))
	}
	client := &http.Client{}
	toCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(toCtx, "GET", query, nil)
	if err != nil {
		return -1, err
	}
	req.Header.Set("Accepts", "application/json")
	req.Header.Add("X-CoinAPI-Key", hdlr.apiKey)

	// send query and receive response
	resp, err := client.Do(req)
	if err != nil {
		return -1, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}
	// extract available credits
	hdlr.credits, _ = strconv.ParseInt(resp.Header.Get("X-RateLimit-Remaining"), 10, 64)
	// parse response
	data := new(CoinapiMarketResponse)
	if err := json.Unmarshal(body, &data); err != nil {
		return -1, err
	}
	return data.Rate, nil
}

// CoinapiMarketResponse is a response from the Market API
type CoinapiMarketResponse struct {
	Time string  `json:"time"`
	Coin string  `json:"asset_id_quote"`
	Fiat string  `json:"asset_id_base"`
	Rate float64 `json:"rate"`
}
