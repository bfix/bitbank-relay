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

package main

import (
	"context"
	"io"
	"net/http"
	"relay/lib"
	"time"

	"github.com/bfix/gospel/logger"
	"github.com/gorilla/mux"
)

// run service
func runService(cfg *lib.ServiceConfig) func(ctx context.Context) error {

	// setup request router
	logger.Println(logger.INFO, "Setting up web service...")
	r := mux.NewRouter()
	r.HandleFunc("/receive/{account}/{coin}", ReceiveHandler)
	r.HandleFunc("/status/{txid}", StatusHandler)

	// assemble HTTP server
	srv := &http.Server{
		Handler:      r,
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

// ReceiveHandler returns an new transaction that includes an (unused) address
// for the given coin and account.
func ReceiveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	io.WriteString(w, `{"alive": true}`)
}

// StatusHandler returns the status for a given transaction
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	io.WriteString(w, `{"alive": true}`)
}

func periodicTasks() {

}
