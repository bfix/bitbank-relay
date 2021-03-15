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
	"os"
	"os/signal"
	"relay/lib"
	"syscall"
	"time"

	"github.com/bfix/gospel/bitcoin/wallet"
	"github.com/bfix/gospel/logger"
)

// Package-global variables
var (
	db  *lib.Database = nil
	cfg *lib.Config   = nil
)

// Application entry point
func main() {
	var err error

	// read configuration
	defer logger.Flush()
	logger.Println(logger.INFO, "Reading configuration...")
	if cfg, err = lib.ReadConfig("config.json"); err != nil {
		logger.Println(logger.ERROR, err.Error())
		return
	}
	// connect to database
	logger.Println(logger.INFO, "Connecting to database...")
	if db, err = lib.Connect(cfg.Db); err != nil {
		logger.Println(logger.ERROR, err.Error())
		return
	}
	defer db.Close()

	// load handlers
	logger.Println(logger.INFO, "Initializing coin handlers:")
	for _, coin := range cfg.Coins {
		_, name := wallet.GetCoinInfo(coin.Symb)
		logger.Printf(logger.INFO, "   * %s (%s)", coin.Symb, name)

		// check if coin is in database
		if _, err = db.GetCoin(coin.Symb); err != nil {
			logger.Println(logger.ERROR, err.Error())
			continue
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
		lib.HdlrList[coin.Symb] = hdlr
	}
	logger.Println(logger.INFO, "Done.")

	// setting up webservice
	srvQuit := runService(cfg.Service)

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
			go periodicTasks()
		}
	}

	// shutdown web service
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	srvQuit(ctx)
}
