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
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bfix/gospel/logger"
)

// GetMarketData returns the current rates for given currencies.
func GetMarketData(ctx context.Context, mdl *Model, fiat string, date int64, coins []string) (map[string]float64, error) {
	// we only have one handler at the moment...
	hdlr, ok := baseMarketHdlrs["coinapi.io"]
	if !ok {
		return nil, fmt.Errorf("no market handler found")
	}
	// check if current or historical rates are requested
	if date < 0 {
		// fetch current rates
		rates, err := hdlr.CurrentRates(ctx, fiat, coins)
		if err != nil {
			return nil, err
		}
		// update rates in coin and rates tables
		logger.Printf(logger.INFO, "Updating market data (%d entries)", len(rates))
		dt := time.Now().Format("2006-01-02")
		for coin, rate := range rates {
			logger.Printf(logger.DBG, "    * %s: %f", coin, rate)
			if err := mdl.UpdateRate(dt, coin, fiat, rate); err != nil {
				logger.Println(logger.ERROR, "UpdateRate: "+err.Error())
			}
		}
		return rates, nil
	}
	// retrieve historical rates
	rates := make(map[string]float64)
	for _, coin := range coins {
		// check rates table first
		dt := time.Unix(date, 0).Format("2006-01-02")
		rate, err := mdl.GetRate(dt, coin, fiat)
		if err != nil {
			logger.Println(logger.ERROR, "GetRate: "+err.Error())
			continue
		}
		if rate < 0 {
			// not in rates table: query market handler.
			if rate, err = hdlr.HistoricalRate(ctx, date, fiat, coin); err != nil {
				logger.Println(logger.ERROR, "HistoricalRate: "+err.Error())
				continue
			}
			// add rate to table
			if err = mdl.SetRate(dt, coin, fiat, rate); err != nil {
				logger.Println(logger.ERROR, "SetRate: "+err.Error())
			}
		}
		rates[coin] = rate
	}
	return rates, nil
}

//======================================================================
// Market handlers
//======================================================================

// MarketHandler retrieves (historical) exchange rates for coins
type MarketHandler interface {
	Init(cfg *MarketHandlerConfig)
	CurrentRates(ctx context.Context, fiat string, coins []string) (map[string]float64, error)
	HistoricalRate(ctx context.Context, date int64, fiat string, coin string) (float64, error)
}

var (
	// map of base market handlers
	baseMarketHdlrs = map[string]MarketHandler{
		"coinapi.io": new(CoinapiMarketHandler),
	}
)

//----------------------------------------------------------------------
// CoinAPI.io
//----------------------------------------------------------------------

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

// CurrentRates returns the current exchange rates for a given list of coins.
func (hdlr *CoinapiMarketHandler) CurrentRates(
	ctx context.Context,
	fiat string,
	coins []string) (map[string]float64, error) {

	// serialize requests
	hdlr.lock.Lock()
	defer hdlr.lock.Unlock()

	// handle all coins at once (current exchange rate)
	query := fmt.Sprintf("https://rest.coinapi.io/v1/exchangerate/%s", fiat)
	client := &http.Client{}
	toCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(toCtx, "GET", query, nil)
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Add("filter_asset_id", strings.Join(coins, ","))
	req.Header.Set("Accepts", "application/json")
	req.Header.Add("X-CoinAPI-Key", hdlr.apiKey)
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
	// extract available credits
	hdlr.credits, _ = strconv.ParseInt(resp.Header.Get("X-RateLimit-Remaining"), 10, 64)

	// parse response
	data := new(CoinapiMarketMultiResponse)
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	// assemble result
	list := make(map[string]float64)
	for _, rate := range data.Rates {
		list[strings.ToLower(rate.Coin)] = 1. / rate.Rate
	}
	return list, nil
}

// HostoricalRate returns the exchange rates for a given date and coin.
func (hdlr *CoinapiMarketHandler) HistoricalRate(
	ctx context.Context,
	date int64,
	fiat string,
	coin string) (float64, error) {

	// serialize requests
	hdlr.lock.Lock()
	defer hdlr.lock.Unlock()

	// assemble query
	query := fmt.Sprintf("https://rest.coinapi.io/v1/exchangerate/%s/%s?time=%s",
		strings.ToUpper(coin), fiat, time.Unix(date, 0).Format("2006-01-02T15:04:05Z"))
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

// CoinapiMarketMultiResponse is a response for mult-coin queries
type CoinapiMarketMultiResponse struct {
	Base  string `json:"asset_id_base"`
	Rates []*struct {
		Time string  `json:"time"`
		Coin string  `json:"asset_id_quote"`
		Rate float64 `json:"rate"`
	} `json:"rates"`
}

// CoinapiMarketResponse is a response from the Market API
type CoinapiMarketResponse struct {
	Time string  `json:"time"`
	Coin string  `json:"asset_id_quote"`
	Fiat string  `json:"asset_id_base"`
	Rate float64 `json:"rate"`
}
