//----------------------------------------------------------------------
// This file is part of 'bitbank-relay'.
// Copyright (C) 2021 Bernd Fix  >Y<
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
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"relay/lib"
	"sort"
	"strings"
	"time"

	"github.com/bfix/gospel/logger"
)

//----------------------------------------------------------------------
// Command-line reporting
//----------------------------------------------------------------------

// Generate reports
func report(args []string) {
	// parse arguments
	flags := flag.NewFlagSet("report", flag.ExitOnError)
	var span, mode, accnt, coin, addr, out, fname string
	flags.StringVar(&span, "r", "*:*", "Date range for report (YYYY-MM-DD)")
	flags.StringVar(&mode, "m", "fast", "Report mode")
	flags.StringVar(&addr, "a", "", "Reported address")
	flags.StringVar(&coin, "c", "", "Reported coin")
	flags.StringVar(&accnt, "p", "", "Reported account")
	flags.StringVar(&out, "o", "csv", "Output format")
	flags.StringVar(&fname, "f", "report.txt", "Output file")
	flags.Parse(args)

	// resolve repository ids
	var (
		coinID, addrID, accntID int64
		err                     error
	)
	if coin != "" {
		if coinID, err = mdl.GetCoinID(coin); err != nil {
			logger.Printf(logger.ERROR, "Invalid coin '%s'\n", coin)
			return
		}
	}
	if accnt != "" {
		if accntID, err = mdl.GetAccountID(accnt); err != nil {
			logger.Printf(logger.ERROR, "Invalid account '%s'\n", coin)
			return
		}
	}
	if addr != "" {
		if addrID, err = mdl.GetAddressID(addr); err != nil {
			logger.Printf(logger.ERROR, "Invalid address '%s'\n", coin)
			return
		}
	}
	// check arguments
	ts := strings.Split(span, ":")
	from, err := convertDate(ts[0], true)
	if err != nil {
		logger.Println(logger.ERROR, "invalid start date: "+err.Error())
		return
	}
	to, err := convertDate(ts[1], false)
	if err != nil {
		logger.Println(logger.ERROR, "invalid end date: "+err.Error())
		return
	}

	// prepare report file
	fOut, err := os.Create(fname)
	if err != nil {
		logger.Println(logger.ERROR, "output file: "+err.Error())
		return
	}
	defer fOut.Close()

	// call report generator.
	ctx := context.Background()
	report, err := doReporting(ctx, addrID, coinID, accntID, from, to, mode, out)
	if err != nil {
		logger.Println(logger.ERROR, "report failed: "+err.Error())
		return
	}
	logger.Printf(logger.DBG, "Report size: %d\n", len(report))
	fOut.Write(report)
	logger.Println(logger.INFO, "Done.")
}

//======================================================================
// Report generator
//======================================================================

// ReportTx represents a fund transaction for a given address
type ReportTx struct {
	Timestamp int64   `json:"timestamp"` // time of transaction
	Account   string  `json:"account"`   // name of receiving account
	Coin      string  `json:"coin"`      // coin label
	Addr      string  `json:"addr"`      // receiving address
	Amount    float64 `json:"amount"`    // received funds
	FiatRecv  float64 `json:"fiatRecv"`  // exchange value at receive time
	FiatNow   float64 `json:"fiatNow"`   // exchange value at report time
}

func doReporting(
	ctx context.Context,
	addrID, coinID, accntID int64, // selection criteria
	from, to int64, // date range for report
	mode, out string,
) (report []byte, err error) {

	// sanity checks.
	if to < from {
		return nil, fmt.Errorf("invalid date range")
	}
	if !strings.Contains(";full;fast;", ";"+mode+";") {
		return nil, fmt.Errorf("invalid report mode")
	}
	if !strings.Contains(";csv;json;html;", ";"+out+";") {
		return nil, fmt.Errorf("invalid output format")
	}
	// list of addresses we care about in the report
	var list []*lib.AddrInfo
	if list, err = mdl.GetAddresses(addrID, accntID, coinID, true); err != nil {
		logger.Println(logger.ERROR, "Failed to collect address list")
		return
	}
	logger.Printf(logger.INFO, "Found %d addresses for reporting.\n", len(list))

	// generate list of transactions for report
	txList := make([]*ReportTx, 0)
	var funds []*lib.Fund
	for _, ai := range list {
		// skip empty address
		if ai.Balance < 1e-8 {
			logger.Printf(logger.INFO, "Skipping empty address '%s'(%s)", ai.Val, ai.CoinSymb)
			continue
		}
		if mode == "fast" {
			// fast mode: only use "incoming" table to build Tx list
			if funds, err = mdl.GetFunds(ai.ID); err != nil {
				logger.Println(logger.ERROR, "Failed to collect funds")
				return
			}
		} else {
			// full mode: retrieve funding transactions from the blockchain
			hdlr, ok := lib.HdlrList[ai.CoinSymb]
			if !ok {
				err = fmt.Errorf("no matching handler for '%s'", ai.CoinName)
				return
			}
			if funds, err = hdlr.GetFunds(ctx, ai.ID, ai.Val); err != nil {
				logger.Printf(logger.ERROR, "tx list failed for '%s'\n", ai.CoinName)
				return
			}
		}
		// convert funds into transactions
		if n := len(funds); n > 0 {
			logger.Printf(logger.INFO, "Found %d funding transactions for %s (%s).\n", n, ai.Val, ai.CoinSymb)
			for _, f := range funds {
				if f.Seen >= from && f.Seen <= to {
					tx := &ReportTx{
						Timestamp: f.Seen,
						Amount:    f.Amount,
						Account:   ai.Account,
						Addr:      ai.Val,
						Coin:      ai.CoinSymb,
					}
					txList = append(txList, tx)
				}
			}
		} else {
			logger.Printf(logger.INFO, "No funding transactions found for '%s'(%s)", ai.Val, ai.CoinSymb)
		}
	}
	logger.Printf(logger.INFO, "Found %d reportable transactions.\n", len(txList))

	// sort list
	sort.Slice(txList, func(i, j int) bool {
		return txList[i].Timestamp < txList[j].Timestamp
	})
	// aggregate data: get fiat value of funds at receive and report time
	logger.Println(logger.INFO, "Aggregating exchange values for funds...")
	for _, tx := range txList {
		// exchange value at receive time
		var rate map[string]float64
		if rate, err = lib.GetMarketData(ctx, mdl, cfg.Handler.Market.Fiat, tx.Timestamp, []string{tx.Coin}); err != nil {
			return
		}
		tx.FiatRecv = tx.Amount * rate[tx.Coin]
		// exchange value at report time
		if rate, err = lib.GetMarketData(ctx, mdl, cfg.Handler.Market.Fiat, -1, []string{tx.Coin}); err != nil {
			return
		}
		tx.FiatNow = tx.Amount * rate[tx.Coin]
	}
	// generate report
	switch out {
	case "json":
		return json.Marshal(txList)
	case "csv":
		wrt := new(bytes.Buffer)
		wrt.WriteString("Date;Account;Amount;Coin;FiatRecv;FiatNow\n")
		for _, tx := range txList {
			fmt.Fprintf(wrt, "%s;\"%s\";%.5f;\"%s\";%.2f;%.2f\n",
				time.Unix(tx.Timestamp, 0).Format("2006-01-02"),
				tx.Account, tx.Amount, tx.Coin, tx.FiatRecv, tx.FiatNow)
		}
		report = wrt.Bytes()
	}
	return
}

//======================================================================
// Helper functions
//======================================================================

// convertDate returns the Unix epoch for a given date (times is 00:00:00
// for start and "23:59:59" for end dates)
func convertDate(d string, isStart bool) (int64, error) {
	if d == "*" {
		if isStart {
			return 0, nil
		}
		return time.Now().Unix(), nil
	}
	t, err := time.Parse("2006-01-02", d)
	if err != nil {
		return -1, err
	}
	if !isStart {
		t = t.Add(24*time.Hour - time.Second)
	}
	return t.Unix(), nil
}
