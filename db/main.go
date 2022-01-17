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
	"fmt"
	"os"
	"relay/lib"

	"github.com/bfix/gospel/logger"
)

var (
	cfg     *lib.Config
	mdl     *lib.Model
	Version string = "v0.0.0"
)

func main() {
	// welcome
	defer logger.Flush()
	logger.Println(logger.INFO, "===============================")
	logger.Println(logger.INFO, "bitbank-relay-db         "+Version)
	logger.Println(logger.INFO, "(c) 2021-2022, Bernd Fix    >Y<")
	logger.Println(logger.INFO, "===============================")

	// parse arguments
	args := os.Args[1:]
	fs := flag.NewFlagSet("main", flag.ExitOnError)
	var (
		confFile string
		export   bool
	)
	fs.BoolVar(&export, "export", false, "Export embedded files")
	fs.StringVar(&confFile, "c", "config.json", "Configuration file (default: config.json)")
	fs.Parse(args)

	// special function "export embedded files"
	if export {
		dir, err := fsys.ReadDir(".")
		if err != nil {
			logger.Println(logger.ERROR, "Export failed: "+err.Error())
			return
		}
		for _, f := range dir {
			fname := f.Name()
			body, err := fsys.ReadFile(fname)
			if err != nil {
				logger.Printf(logger.ERROR, "Export failed (r:%s): %s", fname, err.Error())
				continue
			}
			fOut, err := os.Create(fname)
			if err != nil {
				logger.Printf(logger.ERROR, "Export failed (c:%s): %s", fname, err.Error())
				continue
			}
			if _, err = fOut.Write(body); err != nil {
				logger.Printf(logger.ERROR, "Export failed (w:%s): %s", fname, err.Error())
			}
			fOut.Close()
		}
		return
	}

	// read configuration
	var err error
	logger.Println(logger.INFO, "Reading configuration...")
	if cfg, err = lib.ReadConfigFile(confFile); err != nil {
		logger.Println(logger.ERROR, err.Error())
		return
	}
	// setup logging
	logger.Println(logger.INFO, "Setting up logging...")
	if len(cfg.Service.LogFile) > 0 {
		lfName := fmt.Sprintf(cfg.Service.LogFile, "db")
		logger.LogToFile(lfName)
	}
	logger.SetLogLevelFromName(cfg.Service.LogLevel)

	// connect to model
	logger.Println(logger.INFO, "Connecting to model...")
	if mdl, err = lib.Connect(cfg.Model); err != nil {
		logger.Println(logger.ERROR, err.Error())
		return
	}
	defer mdl.Close()

	// load handlers; assemble list of coin symbols
	logger.Println(logger.INFO, "Initializing coin handlers:")
	if _, err = lib.InitHandler(cfg, mdl); err != nil {
		logger.Println(logger.ERROR, err.Error())
		return
	}
	logger.Println(logger.INFO, "Done.")

	// parse command line arguments (top-level)
	if fs.NArg() == 0 {
		logger.Println(logger.ERROR, "ERROR: No command specified")
		return
	}
	args = fs.Args()
	switch args[0] {
	//------------------------------------------------------------------
	// run gui
	//------------------------------------------------------------------
	case "gui":
		gui(args[1:])

	//------------------------------------------------------------------
	// handle logo methods
	//------------------------------------------------------------------
	case "logo":
		logo(args[1:])
	}
}
