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
	"flag"
	"os"
	"relay/lib"

	"github.com/bfix/gospel/logger"
)

var (
	cfg *lib.Config
	db  *lib.Database
)

func main() {
	// welcome
	defer logger.Flush()
	logger.Println(logger.INFO, "====================================")
	logger.Println(logger.INFO, "bitbank-relay-db v0.1.0 (2021-03-14)")
	logger.Println(logger.INFO, "Copyright (c) 2021, Bernd Fix    >Y<")
	logger.Println(logger.INFO, "====================================")

	// parse arguments
	args := os.Args[1:]
	fs := flag.NewFlagSet("main", flag.ExitOnError)
	var (
		confFile string
	)
	fs.StringVar(&confFile, "c", "config.json", "Configuration file (default: config.json)")
	fs.Parse(args)

	// read configuration
	var err error
	logger.Println(logger.INFO, "Reading configuration...")
	if cfg, err = lib.ReadConfig(confFile); err != nil {
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

	// parse command line arguments (top-level)
	if fs.NArg() == 0 {
		logger.Println(logger.ERROR, "ERROR: No command specified")
		return
	}
	args = fs.Args()
	switch args[0] {
	case "logo":
		logo(args[1:])
	}
}
