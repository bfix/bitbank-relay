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
	"flag"
	"fmt"
	"os"
	"os/signal"
	"relay/lib"
	"syscall"
	"time"

	"github.com/bfix/gospel/logger"
)

// Package-global variables
var (
	db      *lib.Database = nil
	cfg     *lib.Config   = nil
	coins   string        = ""
	Version string        = "v0.0.0"
)

// Application entry point
func main() {
	// welcome
	defer logger.Flush()
	logger.Println(logger.INFO, "==========================")
	logger.Println(logger.INFO, "bitbank-relay-web   "+Version)
	logger.Println(logger.INFO, "(c) 2021, Bernd Fix    >Y<")
	logger.Println(logger.INFO, "==========================")

	// handle command-line arguments
	var confFile string
	flag.StringVar(&confFile, "c", "config.json", "Name of config file (default: ./config.json)")
	flag.Parse()

	// read configuration
	var err error
	defer logger.Flush()
	logger.Println(logger.INFO, "Reading configuration...")
	if cfg, err = lib.ReadConfigFile(confFile); err != nil {
		logger.Println(logger.ERROR, err.Error())
		return
	}
	// setup logging
	if len(cfg.Service.LogFile) > 0 {
		lfName := fmt.Sprintf(cfg.Service.LogFile, "web")
		logger.LogToFile(lfName)
	}
	logger.SetLogLevelFromName(cfg.Service.LogLevel)

	// connect to database
	logger.Println(logger.INFO, "Connecting to database...")
	if db, err = lib.Connect(cfg.Db); err != nil {
		logger.Println(logger.ERROR, err.Error())
		return
	}
	defer db.Close()

	// load handlers; assemble list of coin symbols
	logger.Println(logger.INFO, "Initializing coin handlers:")
	if coins, err = lib.InitHandler(cfg, db); err != nil {
		logger.Println(logger.ERROR, err.Error())
		return
	}
	logger.Println(logger.INFO, "Done.")

	// Prepare context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// setting up balancer service
	balanceCh := lib.StartBalancer(ctx, db, cfg.Balancer)

	// setting up webservice
	srvQuit := runService(cfg.Service)

	// handle OS signals
	sigCh := make(chan os.Signal, 5)
	signal.Notify(sigCh)

	// heart beat
	tick := time.NewTicker(time.Duration(cfg.Service.Epoch) * time.Second)
	epoch := 0

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
			epoch++
			logger.Printf(logger.INFO, "Epoch #%d at %s", epoch, now.String())
			go periodicTasks(ctx, epoch, balanceCh)
		}
	}

	// shutdown web service
	ctxSrv, cancelSrv := context.WithTimeout(ctx, 15*time.Second)
	defer cancelSrv()
	srvQuit(ctxSrv)
}
