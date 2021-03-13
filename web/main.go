//----------------------------------------------------------------------
// This file is part of 'Adresser'.
// Copyright (C) 2021 Bernd Fix >Y<
//
// 'Adresser' is free software: you can redistribute it and/or modify it
// under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// 'Addresser' is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
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
	"addresser/lib"
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bfix/gospel/bitcoin/wallet"
	"github.com/bfix/gospel/logger"
)

// Package-global variables
var (
	handlers             = make(map[string]*lib.Handler)
	db       *Database   = nil
	cfg      *lib.Config = nil
)

// Application entry point
func main() {
	var (
		err   error
		isNew bool
	)
	// read configuration
	defer logger.Flush()
	logger.Println(logger.INFO, "Reading configuration...")
	if cfg, err = lib.ReadConfig("config.json"); err != nil {
		logger.Println(logger.ERROR, err.Error())
		return
	}
	// connect to database
	logger.Println(logger.INFO, "Connecting to database...")
	if db, err = Connect(cfg.Db); err != nil {
		logger.Println(logger.ERROR, err.Error())
		return
	}
	defer db.Close()

	// load handlers
	logger.Println(logger.INFO, "Initializing coin handlers:")
	for _, coin := range cfg.Coins {
		logger.Printf(logger.INFO, "   * %s (%s)", coin.Name, coin.Descr)

		// check if coin is in database
		_, isNew, err = db.GetCoin(coin.Name, coin.Descr)
		if err != nil {
			logger.Println(logger.ERROR, err.Error())
			continue
		}
		if isNew {
			logger.Println(logger.INFO, "     Added to database...")
		}

		// get coin handler
		hdlr, err := lib.NewHandler(coin, wallet.AddrMain)
		if err != nil {
			logger.Println(logger.ERROR, err.Error())
			continue
		}
		// verify handler
		addr, err := hdlr.GetAddress(0)
		if err != nil {
			logger.Println(logger.ERROR, err.Error())
			continue
		}
		if addr != coin.Addr {
			logger.Printf(logger.ERROR, "Addr mismatch: %s != %s", addr, coin.Addr)
			continue
		}
		// save handler
		handlers[coin.Name] = hdlr
	}
	logger.Println(logger.INFO, "Done.")

	// setting up webservice
	ctx, cancel := context.WithCancel(context.Background())
	if err = runService(ctx); err != nil {
		logger.Printf(logger.ERROR, "[gns] RPC failed to start: %s", err.Error())
		return
	}
	defer cancel()

	// handle OS signals
	sigCh := make(chan os.Signal, 5)
	signal.Notify(sigCh)

	// heart beat
	tick := time.NewTicker(5 * time.Minute)

loop:
	for {
		select {
		// handle OS signals
		case sig := <-sigCh:
			switch sig {
			case syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM:
				logger.Printf(logger.INFO, "Terminating service (on signal '%s')\n", sig)
				break loop
			case syscall.SIGHUP:
				logger.Println(logger.INFO, "SIGHUP")
			case syscall.SIGURG:
				// TODO: https://github.com/golang/go/issues/37942
			default:
				logger.Println(logger.INFO, "Unhandled signal: "+sig.String())
			}
		// handle heart beat
		case now := <-tick.C:
			logger.Println(logger.INFO, "Heart beat at "+now.String())
			go periodicTasks(ctx)
		}
	}
}
