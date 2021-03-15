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
	"encoding/base64"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/bfix/gospel/logger"
)

func logo(args []string) {
	if len(args) == 0 {
		logger.Println(logger.ERROR, "ERROR: logo: No sub-command specified")
		logger.Println(logger.INFO, "logo sub-commands: 'import','list'")
		return
	}
	switch args[0] {
	case "import":
		logo_import(args[1:])
	}
}

func logo_import(args []string) {
	fs := flag.NewFlagSet("logo_import", flag.ExitOnError)
	var (
		dir string
	)
	fs.StringVar(&dir, "i", "", "Folder with coin logos")
	fs.Parse(args)

	if len(dir) == 0 {
		logger.Println(logger.ERROR, "ERROR: logo-import -- missing input folder")
		fs.Usage()
		return
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		logger.Println(logger.ERROR, "ERROR: "+err.Error())
		return
	}
	for _, f := range files {
		fname := filepath.Join(dir, f.Name())
		if !strings.HasSuffix(fname, ".svg") {
			continue
		}
		in, err := os.Open(fname)
		if err != nil {
			logger.Println(logger.ERROR, "ERROR: "+err.Error())
			continue
		}
		defer in.Close()
		body, err := ioutil.ReadAll(in)
		if err != nil {
			logger.Println(logger.ERROR, "ERROR: "+err.Error())
			continue
		}
		logo := base64.StdEncoding.EncodeToString(body)
		base := filepath.Base(fname)
		coin := base[:len(base)-4]

		logger.Printf(logger.INFO, "Adding logo for coin '%s'\n", coin)
		if err = db.SetCoinLogo(coin, logo); err != nil {
			logger.Println(logger.ERROR, "ERROR: "+err.Error())
		}
	}
}
