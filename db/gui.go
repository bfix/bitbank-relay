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
	"flag"
	"fmt"
	"io"
	"net/http"
	"relay/lib"
	"text/template"
	"time"

	"github.com/bfix/gospel/logger"
)

var (
	tpl *template.Template // HTML templates
	srv *http.Server       // HTTP server
)

// Start the GUI for database management and relay maintenance
func gui(args []string) {
	// parse arguments
	fs := flag.NewFlagSet("gui", flag.ExitOnError)
	var (
		listen string
	)
	fs.StringVar(&listen, "l", "localhost:8080", "Listen address for web GUI")
	fs.Parse(args)

	// read and prepare templates
	tpl = template.New("gui")
	tpl.Funcs(template.FuncMap{
		"mul": func(a, b float64) string {
			return fmt.Sprintf("%.02f", a*b)
		},
		"trim": func(a float64) string {
			return fmt.Sprintf("%.08f", a)
		},
	})
	if _, err := tpl.ParseFiles("gui.htpl"); err != nil {
		logger.Println(logger.ERROR, "GUI templates: "+err.Error())
		return
	}

	// define request handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/coin/", coinHandler)
	mux.HandleFunc("/account/", accountHandler)
	mux.HandleFunc("/addr/", addressHandler)
	mux.HandleFunc("/tx/", transactionHandler)
	mux.HandleFunc("/close/", closeHandler)
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
	Fiat      string             // name of the fiat currency to use
	Coins     []*lib.AccCoinInfo // list of active coins
	Accounts  []*lib.AccntInfo   // list of active accounts
	Addresses []*lib.AddrInfo    // list of (active) addresses
}

func guiHandler(w http.ResponseWriter, r *http.Request) {
	// collect information for the dashboard
	dd := new(DashboardData)
	dd.Fiat = cfg.Market.Fiat

	// collect coin info
	var err error
	if dd.Coins, err = db.GetAccumulatedCoins(); err != nil {
		io.WriteString(w, "ERROR: "+err.Error())
		return
	}
	// collect account info
	if dd.Accounts, err = db.GetAccounts(); err != nil {
		io.WriteString(w, "ERROR: "+err.Error())
		return
	}
	// collect address info
	if dd.Addresses, err = db.GetAddresses(); err != nil {
		io.WriteString(w, "ERROR: "+err.Error())
		return
	}
	// show dashboard
	renderPage(w, dd, "dashboard")
}

//======================================================================
// handle coin-related GUI requests
//======================================================================

func coinHandler(w http.ResponseWriter, r *http.Request) {
}

//======================================================================
// handle account-related GUI requests
//======================================================================

func accountHandler(w http.ResponseWriter, r *http.Request) {

}

//======================================================================
// handle address-related GUI requests
//======================================================================

func addressHandler(w http.ResponseWriter, r *http.Request) {

}

//======================================================================
// handle transaction-related GUI requests
//======================================================================

func transactionHandler(w http.ResponseWriter, r *http.Request) {

}

//======================================================================
// handle coin-related GUI requests
//======================================================================

func closeHandler(w http.ResponseWriter, r *http.Request) {
	// close the server
	srv.Close()
	io.WriteString(w, "Bye")
}

//======================================================================
//======================================================================

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
