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
	"regexp"
	"relay/lib"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/bfix/gospel/logger"
)

//go:embed gui.htpl
var fsys embed.FS

var (
	tpl *template.Template // HTML templates
	srv *http.Server       // HTTP server
)

// Start the GUI for model management and relay maintenance
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
	if _, err := tpl.ParseFS(fsys, "gui.htpl"); err != nil {
		logger.Println(logger.ERROR, "GUI templates: "+err.Error())
		return
	}

	// define request handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/coin/", coinHandler)
	mux.HandleFunc("/account/", accountHandler)
	mux.HandleFunc("/addr/", addressHandler)
	mux.HandleFunc("/new/", newHandler)
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
	Incoming  []*lib.Incoming    `json:"incoming"`  // list of recently incoming funds
	Coins     []*lib.AccCoinInfo `json:"coins"`     // list of active coins
	Accounts  []*lib.AccntInfo   `json:"accounts"`  // list of active accounts
	Addresses []*lib.AddrInfo    `json:"addresses"` // list of (active) addresses
}

// handle dashboard (main entry page)
func guiHandler(w http.ResponseWriter, r *http.Request) {
	// collect information for the dashboard
	dd := new(DashboardData)
	dd.Fiat = cfg.Handler.Market.Fiat

	// collect coin info
	var err error
	if dd.Coins, err = mdl.GetAccumulatedCoin(0); err != nil {
		io.WriteString(w, "ERROR: "+err.Error())
		return
	}
	// collect account info
	if dd.Accounts, err = mdl.GetAccounts(0); err != nil {
		io.WriteString(w, "ERROR: "+err.Error())
		return
	}
	// collect address info
	if dd.Addresses, err = mdl.GetAddresses(0, 0, 0, false); err != nil {
		io.WriteString(w, "ERROR: "+err.Error())
		return
	}
	// collect list of recently received funds
	if dd.Incoming, err = mdl.ListIncoming(25); err != nil {
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
	cd.Fiat = cfg.Handler.Market.Fiat

	if id, ok := queryInt(query, "id"); ok {
		// check if we switch assignments
		if accept := query.Get("accept"); len(accept) > 0 {
			on, off, err := parseOnOffList(accept)
			if err != nil {
				logger.Println(logger.ERROR, "coinHandler: "+err.Error())
				return
			}
			for _, accnt := range on {
				if err := mdl.ChangeAssignment(id, accnt, true); err != nil {
					return
				}
			}
			for _, accnt := range off {
				if err := mdl.ChangeAssignment(id, accnt, false); err != nil {
					return
				}
			}
			// do a redirect after switching assignments
			http.Redirect(w, r, fmt.Sprintf("/coin/?id=%d", id), http.StatusFound)
			return
		}
		// get assignments from model
		if res, err := mdl.GetAccumulatedCoin(id); err == nil {
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
	ad.Fiat = cfg.Handler.Market.Fiat

	if id, ok := queryInt(query, "id"); ok {
		// check if we switch assignments
		if accept := query.Get("accept"); len(accept) > 0 {
			on, off, err := parseOnOffList(accept)
			if err != nil {
				logger.Println(logger.ERROR, "accountHandler: "+err.Error())
				return
			}
			for _, coin := range on {
				if err := mdl.ChangeAssignment(coin, id, true); err != nil {
					return
				}
			}
			for _, coin := range off {
				if err := mdl.ChangeAssignment(coin, id, false); err != nil {
					return
				}
			}
			// do a redirect after switch assignments
			http.Redirect(w, r, fmt.Sprintf("/account/?id=%d", id), http.StatusFound)
			return
		}
		// get assignments from model
		if res, err := mdl.GetAccounts(id); err == nil {
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
	Mode    int               `json:"mode"`    // selection mode
	Account string            `json:"account"` // account name
	Coin    string            `json:"coin"`    // coin name
	Fiat    string            `json:"fiat"`    // fiat currency
	Addrs   []*lib.AddrInfo   `json:"addrs"`   // info about addresses
	Links   map[string]string `json:"links"`   // links
}

// handle "address" page
func addressHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	// show address info
	query := r.URL.Query()
	ad := new(AddressData)
	ad.Fiat = cfg.Handler.Market.Fiat
	ad.Links = make(map[string]string)

	if id, ok := queryInt(query, "id"); ok {
		// check for special actions like "close"
		if mode := query.Get("m"); len(mode) > 0 {
			var err error
			switch mode {
			// close address for further use
			case "close":
				err = mdl.CloseAddress(id)
			// lock address after spending
			case "lock":
				err = mdl.LockAddress(id)
			// flag address for balance sync
			case "sync":
				err = mdl.SyncAddress(id)
			}
			if err != nil {
				logger.Printf(logger.ERROR, "addressHandler: "+err.Error())
			}
			// redirect to address page (id-view)
			http.Redirect(w, r, fmt.Sprintf("/addr/?id=%d", id), http.StatusFound)
		}
		// normal address selection
		ad.Addrs, err = mdl.GetAddresses(id, 0, 0, true)
		if len(ad.Addrs) == 0 {
			ad.Mode = 0
		} else {
			ad.Mode = 1
			ad.Account = ad.Addrs[0].Account
			ad.Coin = ad.Addrs[0].CoinName
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
		ad.Addrs, err = mdl.GetAddresses(0, accntId, coinId, true)
		if len(ad.Addrs) == 0 {
			ad.Mode = 0
		} else {
			ad.Mode = 2
			ad.Account = "*"
			if accntId != 0 {
				ad.Account = ad.Addrs[0].Account
			}
			ad.Coin = "*"
			if coinId != 0 {
				ad.Coin = ad.Addrs[0].CoinName
			}
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
	Mode    int                `json:"mode"`    // 0=all, 1=addr, 2=account, 3=coin
	Address string             `json:"address"` // address string
	Account string             `json:"account"` // account name
	Coin    string             `json:"coin"`    // coin name
	Txs     []*lib.Transaction `json:"txs"`     // list of transactions
	Links   map[string]string  `json:"links"`   // links
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
	}
	if accnt, ok = queryInt(query, "accnt"); ok {
		td.Mode = 2
		td.Links["&#9654; Account"] = fmt.Sprintf("/account/?id=%d", accnt)
	}
	if coin, ok = queryInt(query, "coin"); ok {
		if td.Mode == 0 {
			td.Mode = 3
		} else if td.Mode == 2 {
			td.Mode = 4
		}
		td.Links["&#9654; Coin"] = fmt.Sprintf("/coin/?id=%d", coin)
	}
	if td.Txs, err = mdl.GetTransactions(addr, accnt, coin); err != nil {
		logger.Println(logger.ERROR, "txHandler: "+err.Error())
		return
	}
	// set page title
	if td.Txs != nil && len(td.Txs) > 0 {
		td.Address = td.Txs[0].Addr
		td.Account = td.Txs[0].Accnt
		td.Coin = td.Txs[0].Coin
	}
	// provide fallback for empty link list
	if len(td.Links) == 0 {
		td.Links["Home"] = "/"
	}
	// show address page
	renderPage(w, td, "tx")
}

//======================================================================
// Create a new element (account)
//======================================================================

// NewData holds the data needed to render a "Create new ..." dialog
type NewData struct {
	Mode string `json:"mode"` // kind of object to be created
}

func newHandler(w http.ResponseWriter, r *http.Request) {
	// GET requests initiate a "new" dialog
	if r.Method == "GET" {
		nd := new(NewData)
		switch r.URL.Query().Get("m") {
		// create new account
		case "accnt":
			nd.Mode = "accnt"
		}
		// show address page
		renderPage(w, nd, "new")
		return
	}
	// get POST parameters
	if err := r.ParseForm(); err != nil {
		logger.Printf(logger.ERROR, "newHandler: %v", err)
		return
	}
	switch r.FormValue("mode") {

	// create new account object
	case "accnt":
		label := r.FormValue("label")
		if len(label) == 0 || !checkChars(label, "^[A-Za-z0-9_]*$") {
			logger.Println(logger.ERROR, "newAccount: Invalid label")
			return
		}
		name := r.FormValue("name")
		if len(name) == 0 {
			logger.Println(logger.ERROR, "newAccount: Invalid name")
			return
		}
		if err := mdl.NewAccount(label, name); err != nil {
			logger.Printf(logger.ERROR, "newAccount: %v", err)
			return
		}
	}
	// redirect back to main page
	http.Redirect(w, r, "/", http.StatusFound)
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
	// save logo to model
	if err := mdl.SetCoinLogo(coin, logo); err != nil {
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

// check if all chars in "str" match pattern
func checkChars(str, pattern string) bool {
	match, _ := regexp.MatchString(pattern, str)
	return match
}
