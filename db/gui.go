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
	"embed"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"relay/lib"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/bfix/gospel/logger"
)

//go:embed gui.htpl
var fs embed.FS

var (
	tpl *template.Template // HTML templates
	srv *http.Server       // HTTP server
)

// Start the GUI for database management and relay maintenance
func gui(args []string) {
	// parse arguments
	flags := flag.NewFlagSet("gui", flag.ExitOnError)
	var (
		listen string
	)
	flags.StringVar(&listen, "l", "localhost:8080", "Listen address for web GUI")
	flags.Parse(args)

	// read and prepare templates
	tpl = template.New("gui")
	tpl.Funcs(template.FuncMap{
		"mul": func(a, b float64) float64 {
			return a * b
		},
		"trim": func(a float64, b int) string {
			return fmt.Sprintf("%.[2]*[1]f", a, b)
		},
		"valid": func(a interface{}) bool {
			return a != nil
		},
		"date": func(ts int64) string {
			return time.Unix(ts, 0).Format("02 Jan 06 15:04")
		},
	})
	if _, err := tpl.ParseFS(fs, "gui.htpl"); err != nil {
		logger.Println(logger.ERROR, "GUI templates: "+err.Error())
		return
	}

	// define request handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/coin/", coinHandler)
	mux.HandleFunc("/account/", accountHandler)
	mux.HandleFunc("/addr/", addressHandler)
	mux.HandleFunc("/logo/", logoHandler)
	mux.HandleFunc("/tx/", transactionHandler)
	mux.HandleFunc("/", guiHandler)

	// prepare HTTP server
	srv = &http.Server{
		Addr:              listen,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       300 * time.Second,
		ReadHeaderTimeout: 20 * time.Second,
		Handler:           mux,
	}
	// run HTTP server
	logger.Printf(logger.INFO, "Starting HTTP server at %s...", listen)
	if err := srv.ListenAndServe(); err != nil {
		logger.Println(logger.ERROR, "GUI listener: "+err.Error())
	}
}

//======================================================================
// handle basic GUI request (dashboard)
//======================================================================

// DashboardData holds all information to render the dashboard view.
type DashboardData struct {
	Fiat      string             `json:"fiat"`      // name of the fiat currency to use
	Coins     []*lib.AccCoinInfo `json:"coins"`     // list of active coins
	Accounts  []*lib.AccntInfo   `json:"accounts"`  // list of active accounts
	Addresses []*lib.AddrInfo    `json:"addresses"` // list of (active) addresses
}

// handle dashboard (main entry page)
func guiHandler(w http.ResponseWriter, r *http.Request) {
	// collect information for the dashboard
	dd := new(DashboardData)
	dd.Fiat = cfg.Market.Fiat

	// collect coin info
	var err error
	if dd.Coins, err = db.GetAccumulatedCoin(0); err != nil {
		io.WriteString(w, "ERROR: "+err.Error())
		return
	}
	// collect account info
	if dd.Accounts, err = db.GetAccounts(0); err != nil {
		io.WriteString(w, "ERROR: "+err.Error())
		return
	}
	// collect address info
	if dd.Addresses, err = db.GetAddresses(0, 0, 0, false); err != nil {
		io.WriteString(w, "ERROR: "+err.Error())
		return
	}
	// show dashboard
	renderPage(w, dd, "dashboard")
}

//======================================================================
// handle coin-related GUI requests
//======================================================================

// CoinData holds the information needed to render a coin page
type CoinData struct {
	Fiat string           `json:"fiat"` // fiat currency
	Coin *lib.AccCoinInfo `json:"coin"` // info about coin
}

// process "coin" page request
func coinHandler(w http.ResponseWriter, r *http.Request) {
	// show coin info
	query := r.URL.Query()
	cd := new(CoinData)
	cd.Fiat = cfg.Market.Fiat

	if id, ok := queryInt(query, "id"); ok {
		// check if we switch assignments
		if accept := query.Get("accept"); len(accept) > 0 {
			on, off, err := parseOnOffList(accept)
			if err != nil {
				logger.Println(logger.ERROR, "coinHandler: "+err.Error())
				return
			}
			for _, accnt := range on {
				if err := db.ChangeAssignment(id, accnt, true); err != nil {
					return
				}
			}
			for _, accnt := range off {
				if err := db.ChangeAssignment(id, accnt, false); err != nil {
					return
				}
			}
			// do a redirect after switching assignments
			http.Redirect(w, r, fmt.Sprintf("/coin/?id=%d", id), http.StatusFound)
			return
		}
		// get assignments from database
		if res, err := db.GetAccumulatedCoin(id); err == nil {
			if len(res) > 0 {
				cd.Coin = res[0]
			} else {
				logger.Println(logger.WARN, "coinHandler: no coin infos")
				return
			}
		} else {
			logger.Println(logger.ERROR, "coinHandler: "+err.Error())
			return
		}
	} else {
		logger.Println(logger.WARN, "coinHandler: No ID in query")
		return
	}
	// show coin page
	renderPage(w, cd, "coin")
}

//======================================================================
// handle account-related GUI requests
//======================================================================

// AccountData holds the information needed to render an "account" page.
type AccountData struct {
	Fiat  string         `json:"fiat"`  // fiat currency
	Accnt *lib.AccntInfo `json:"accnt"` // info about account
}

// handle "account" page
func accountHandler(w http.ResponseWriter, r *http.Request) {
	// show account info
	query := r.URL.Query()
	ad := new(AccountData)
	ad.Fiat = cfg.Market.Fiat

	if id, ok := queryInt(query, "id"); ok {
		// check if we switch assignments
		if accept := query.Get("accept"); len(accept) > 0 {
			on, off, err := parseOnOffList(accept)
			if err != nil {
				logger.Println(logger.ERROR, "accountHandler: "+err.Error())
				return
			}
			for _, coin := range on {
				if err := db.ChangeAssignment(coin, id, true); err != nil {
					return
				}
			}
			for _, coin := range off {
				if err := db.ChangeAssignment(coin, id, false); err != nil {
					return
				}
			}
			// do a redirect after switch assignments
			http.Redirect(w, r, fmt.Sprintf("/account/?id=%d", id), http.StatusFound)
			return
		}
		// get assignments from database
		if res, err := db.GetAccounts(id); err == nil {
			if len(res) > 0 {
				ad.Accnt = res[0]
			} else {
				logger.Println(logger.WARN, "accountHandler: no account infos")
				return
			}
		} else {
			logger.Println(logger.ERROR, "accountHandler: "+err.Error())
			return
		}
	} else {
		logger.Println(logger.WARN, "accountHandler: No ID in query")
		return
	}
	// show account page
	renderPage(w, ad, "account")
}

//======================================================================
// handle address-related GUI requests
//======================================================================

// AddressData holds the information needed to render an "address" page.
type AddressData struct {
	Title string            `json:"title"` // title for collection
	Fiat  string            `json:"fiat"`  // fiat currency
	Addrs []*lib.AddrInfo   `json:"addrs"` // info about addresses
	Links map[string]string `json:"links"` // links
}

// handle "address" page
func addressHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	// show address info
	query := r.URL.Query()
	ad := new(AddressData)
	ad.Fiat = cfg.Market.Fiat
	ad.Links = make(map[string]string)

	if id, ok := queryInt(query, "id"); ok {
		ad.Addrs, err = db.GetAddresses(id, 0, 0, true)
		if len(ad.Addrs) == 0 {
			ad.Title = "No address(es) found..."
		} else {
			accnt := ad.Addrs[0].Account
			coin := ad.Addrs[0].Coin
			ad.Title = fmt.Sprintf("Address for '%s' (%s)", accnt, coin)
		}
	} else {
		accntId, _ := queryInt(query, "accnt")
		coinId, _ := queryInt(query, "coin")
		if accntId != 0 {
			ad.Links["&#9654; Account"] = fmt.Sprintf("/account/?id=%d", accntId)
		}
		if coinId != 0 {
			ad.Links["&#9654; Coin"] = fmt.Sprintf("/coin/?id=%d", coinId)
		}
		ad.Addrs, err = db.GetAddresses(0, accntId, coinId, true)
		if len(ad.Addrs) == 0 {
			ad.Title = "No address(es) found..."
		} else {
			accnt := "*"
			if accntId != 0 {
				accnt = ad.Addrs[0].Account
			}
			coin := "*"
			if coinId != 0 {
				coin = ad.Addrs[0].Coin
			}
			ad.Title = fmt.Sprintf("Address(es) for %s (%s)", accnt, coin)
		}
	}
	if err != nil {
		logger.Println(logger.ERROR, "addressHandler: "+err.Error())
		return
	}
	// provide fallback for empty link list
	if len(ad.Links) == 0 {
		ad.Links["Home"] = "/"
	}
	// show address page
	renderPage(w, ad, "address")
}

//======================================================================
// transaction handler
//======================================================================

// TxData holds information needed to rended a transaction page
type TxData struct {
	Title    string             `json:"title"`    // page title
	SubTitle string             `json:"subTitle"` // page subtitle
	Mode     int                `json:"mode"`     // 0=all, 1=addr, 2=account, 3=coin
	Txs      []*lib.Transaction `json:"txs"`      // list of transactions
	Links    map[string]string  `json:"links"`    // links
}

// handle transaction requests
func transactionHandler(w http.ResponseWriter, r *http.Request) {
	// show transaction infos
	var (
		err               error
		addr, accnt, coin int64
		ok                bool
	)
	query := r.URL.Query()
	td := new(TxData)
	td.Mode = 0
	td.Links = make(map[string]string)

	// get transaction based on query parameters
	if addr, ok = queryInt(query, "addr"); ok {
		td.Mode = 1
		td.Links["&#9654; Address"] = fmt.Sprintf("/addr/?id=%d", addr)
	} else if accnt, ok = queryInt(query, "accnt"); ok {
		td.Mode = 2
		td.Links["&#9654; Account"] = fmt.Sprintf("/account/?id=%d", accnt)
	}
	if coin, ok = queryInt(query, "coin"); ok {
		if td.Mode == 0 {
			td.Mode = 3
		}
		td.Links["&#9654; Coin"] = fmt.Sprintf("/coin/?id=%d", coin)
	}
	if td.Txs, err = db.GetTransactions(addr, accnt, coin); err != nil {
		logger.Println(logger.ERROR, "txHandler: "+err.Error())
		return
	}
	// set page title
	td.Title = "No transactions found..."
	if td.Txs != nil && len(td.Txs) > 0 {
		addr := td.Txs[0].Addr
		accnt := td.Txs[0].Accnt
		coin := td.Txs[0].Coin
		switch td.Mode {
		case 0:
			td.Title = "All transactions"
		case 1:
			td.Title = fmt.Sprintf("Transactions for '%s'", addr)
			td.SubTitle = fmt.Sprintf("%s: '%s'", coin, accnt)
		case 2:
			td.Title = fmt.Sprintf("Transactions for '%s'", accnt)
		case 3:
			td.Title = fmt.Sprintf("Transactions for '%s'", coin)
		}
	}
	// provide fallback for empty link list
	if len(td.Links) == 0 {
		td.Links["Home"] = "/"
	}
	// show address page
	renderPage(w, td, "tx")
}

//======================================================================
// handle upload of new coin logo
//======================================================================

func logoHandler(w http.ResponseWriter, r *http.Request) {
	// get POST parameters
	if err := r.ParseMultipartForm(0); err != nil {
		logger.Printf(logger.ERROR, "ParseForm() err: %v", err)
		return
	}
	id := r.FormValue("id")
	coin := r.FormValue("coin")
	file, _, err := r.FormFile("logo")
	if err != nil {
		logger.Printf(logger.ERROR, "ParseForm() err: %v", err)
		return
	}
	defer file.Close()
	// get logo data
	body, err := ioutil.ReadAll(file)
	if err != nil {
		logger.Printf(logger.ERROR, "ParseForm() err: %v", err)
		return
	}
	logo := base64.StdEncoding.EncodeToString(body)
	// save logo to database
	if err := db.SetCoinLogo(coin, logo); err != nil {
		logger.Printf(logger.ERROR, "ParseForm() err: %v", err)
		return
	}
	// redirect back to coin page
	http.Redirect(w, r, "/coin/?id="+id, http.StatusFound)
}

//======================================================================
// Helper methods
//======================================================================

// render a webpage with given data and template reference
func renderPage(w io.Writer, data interface{}, page string) {
	// create content section
	t := tpl.Lookup(page)
	if t == nil {
		io.WriteString(w, "No template '"+page+"' found")
		return
	}
	content := new(bytes.Buffer)
	if err := t.Execute(content, data); err != nil {
		io.WriteString(w, err.Error())
		return
	}
	// emit final page
	t = tpl.Lookup("main")
	if t == nil {
		io.WriteString(w, "No main template found")
		return
	}
	if err := t.Execute(w, content.String()); err != nil {
		io.WriteString(w, err.Error())
	}
}

// parse an on/off list of form "id1,id2,id3|id4,id5" and return two lists
// of integers
func parseOnOffList(list string) (on, off []int64, err error) {
	parse := func(s string) (list []int64, err error) {
		if len(s) == 0 {
			return
		}
		for _, elem := range strings.Split(s, ",") {
			var val int64
			if val, err = strconv.ParseInt(elem, 10, 64); err != nil {
				return
			}
			list = append(list, val)
		}
		return
	}
	parts := strings.Split(list, "|")
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("parseOnOffList")
	}
	if on, err = parse(parts[0]); err != nil {
		return
	}
	off, err = parse(parts[1])
	return
}

// return an integer URL query value
func queryInt(query url.Values, key string) (int64, bool) {
	if id := query.Get(key); len(id) > 0 {
		if val, err := strconv.ParseInt(id, 10, 64); err == nil {
			return val, true
		}
	}
	return 0, false
}
