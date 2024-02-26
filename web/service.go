//----------------------------------------------------------------------
// This file is part of 'bitbank-relay'.
// Copyright (C) 2021-2024, Bernd Fix >Y<
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
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"relay/lib"
	"time"

	"github.com/bfix/gospel/logger"
	qrcode "github.com/yeqown/go-qrcode"
)

//----------------------------------------------------------------------
// run service
//----------------------------------------------------------------------

func runService(cfg *lib.ServiceConfig) func(ctx context.Context) error {

	// setup request router
	logger.Println(logger.INFO, "Setting up web service...")
	mux := http.NewServeMux()
	mux.HandleFunc("/list/", listHandler)
	mux.HandleFunc("/receive/", receiveHandler)
	mux.HandleFunc("/status/", statusHandler)

	// assemble HTTP server
	srv := &http.Server{
		Handler:      mux,
		Addr:         cfg.Listen,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	// start server
	logger.Println(logger.INFO, "Waiting for client requests...")
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.Println(logger.ERROR, err.Error())
		}
	}()
	return srv.Shutdown
}

//----------------------------------------------------------------------
// ListHandler returns a list of coins accepted for a given account.
// Returns an empty list if no valid account is specified.
//----------------------------------------------------------------------

func listHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	accnt := r.FormValue("a")
	if len(accnt) == 0 {
		logger.Println(logger.INFO, "List[0]: no account")
		io.WriteString(w, "[]")
		return
	}
	list, err := mdl.GetCoins(accnt)
	if err != nil {
		logger.Println(logger.ERROR, "List[1]: "+err.Error())
		io.WriteString(w, "[]")
		return
	}
	body, err := json.Marshal(list)
	if err != nil {
		logger.Println(logger.ERROR, "List[2]: "+err.Error())
		io.WriteString(w, "[]")
		return
	}
	w.Write(body)
}

//----------------------------------------------------------------------
// ReceiveHandler returns an new transaction that includes an (unused) address
// for the given coin and account.
//----------------------------------------------------------------------

type txResponse struct {
	Error string           `json:"error,omitempty"`
	Tx    *lib.Transaction `json:"tx"`
	Qr    string           `json:"qr"`
	Coin  *lib.CoinInfo    `json:"coin"`
}

func receiveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// create response and send it on exit
	resp := new(txResponse)
	defer func() {
		buf, _ := json.Marshal(resp)
		w.Write(buf)
	}()

	// get address for given account and coin
	accnt := r.FormValue("a")
	coin := r.FormValue("c")
	tx, err := mdl.NewTransaction(coin, accnt)
	if err != nil {
		logger.Printf(logger.ERROR, "receive: account=%s, coin=%s failed: %s\n", accnt, coin, err.Error())
		resp.Error = err.Error()
		return
	}
	logger.Printf(logger.INFO, "receive: account=%s, coin=%s => %s\n", accnt, coin, tx.Addr)

	// generate QR code of address
	qr := "data:image/jpeg;base64,"
	qrc, err := qrcode.New(tx.Addr)
	if err == nil {
		buf := new(bytes.Buffer)
		qrc.SaveTo(buf)
		qr += base64.StdEncoding.EncodeToString(buf.Bytes())
	} else {
		qr = ""
	}
	// get coin info
	ci, err := mdl.GetCoin(coin)
	if err != nil {
		resp.Error = err.Error()
		return
	}
	// assemble response
	resp.Qr = qr
	resp.Tx = tx
	resp.Coin = ci
}

//----------------------------------------------------------------------
// StatusHandler returns the status for a given transaction
//----------------------------------------------------------------------

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// create response and send it on exit
	resp := new(txResponse)
	defer func() {
		buf, _ := json.Marshal(resp)
		w.Write(buf)
	}()

	// get transaction
	var err error
	tx := r.FormValue("t")
	logger.Printf(logger.DBG, "status: tx=%s\n", tx)

	if resp.Tx, err = mdl.GetTransaction(tx); err != nil {
		resp.Error = err.Error()
		return
	}
	// generate QR code of address
	qr := "data:image/jpeg;base64,"
	qrc, err := qrcode.New(resp.Tx.Addr)
	if err == nil {
		buf := new(bytes.Buffer)
		qrc.SaveTo(buf)
		qr += base64.StdEncoding.EncodeToString(buf.Bytes())
	} else {
		qr = ""
	}
	// get coin info
	ci, err := mdl.GetCoin(resp.Tx.Coin)
	if err != nil {
		resp.Error = err.Error()
		return
	}
	// assemble response
	resp.Qr = qr
	resp.Coin = ci
}
