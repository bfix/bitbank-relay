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
	"bufio"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"relay/lib"

	"github.com/bfix/gospel/bitcoin/wallet"
	"github.com/bfix/gospel/logger"
)

//go:embed config-template.json
var fsys embed.FS

func main() {
	// parse and process command-line options
	var (
		network string
		inConf  string
		outConf string
		export  bool
	)
	flag.BoolVar(&export, "export", false, "Export embedded files")
	flag.StringVar(&network, "n", "main", "Network [main|test|reg]")
	flag.StringVar(&inConf, "i", "", "Configuration template file (default: embedded config)")
	flag.StringVar(&outConf, "o", "config.json", "Configuration output file (default: config.json)")
	flag.Parse()

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

	// Ask for passphrase
	// N.B.: This is not a BIP39 password added to the list of seed words,
	// but a passphrase used to generate the seed words for a BIP39 wallet.
	fmt.Printf(">>> Passphrase: ")
	rdr := bufio.NewReader(os.Stdin)
	in, _, err := rdr.ReadLine()
	if err != nil {
		fmt.Println("<<< ERROR: " + err.Error())
		return
	}
	// compute entropy, seed words and seed value
	ent := sha256.Sum256(in)
	words, err := wallet.EntropyToWords(ent[:])
	if err != nil {
		fmt.Println("<<< ERROR: " + err.Error())
		return
	}
	seed, _ := wallet.WordsToSeed(words, "")

	// output computed information
	fmt.Printf("<<<    Entropy: %s\n", hex.EncodeToString(ent[:]))
	fmt.Printf("<<<       Seed: %s\n", hex.EncodeToString(seed))
	fmt.Println("<<<------------------------------------------------")
	fmt.Println("<<< Seed words:")
	n := len(words) / 2
	for i := 0; i < n; i++ {
		fmt.Printf("<<<    %2d: %-20s %2d: %-20s\n", i+1, words[i], i+n+1, words[i+n])
	}
	fmt.Println("<<<------------------------------------------------")

	// create a HD wallet for the given seed
	hd := wallet.NewHD(seed)
	pk := hd.MasterPublic()
	fmt.Printf("<<< Master Pub: %s\n", pk)
	sk := hd.MasterPrivate()
	fmt.Printf("<<< Master Prv: %s\n", sk)

	// load config template
	fmt.Println("<<< Generate configuration file...")
	var cfg *lib.Config
	if len(inConf) > 0 {
		cfg, err = lib.ReadConfigFile(inConf)
	} else {
		var f fs.File
		if f, err = fsys.Open("config-template.json"); err == nil {
			cfg, err = lib.ReadConfig(f)
		}
	}
	if err != nil {
		fmt.Println("<<< ERROR: " + err.Error())
		return
	}
	// process all entries
	netw := lib.GetNetwork(network)
	for _, coin := range cfg.Coins {
		fmt.Printf("<<<    Processing '%s'...\n", coin.Symb)

		// get base extended public key for given account
		bpk, err := hd.Public(coin.Path)
		if err != nil {
			fmt.Println("<<< ERROR: " + err.Error())
			continue
		}
		bpk.Data.Version = coin.GetXDVersion()
		coin.Pk = bpk.String()

		// get coin handler
		hdlr, err := lib.NewHandler(coin, netw)
		if err != nil {
			fmt.Println("<<< ERROR: " + err.Error())
			continue
		}

		// compute addresses; save first for check
		for idx := 0; idx < 10; idx++ {
			addr, err := hdlr.GetAddress(idx)
			if err != nil {
				fmt.Println("<<< ERROR: " + err.Error())
				continue
			}
			if idx == 0 {
				coin.Addr = addr
			}
			fmt.Printf("<<<    %2d: %s\n", idx, addr)
		}
	}
	// save to configuration file
	if err = lib.WriteConfigFile(outConf, cfg); err != nil {
		fmt.Println("<<< ERROR: " + err.Error())
		return
	}
	fmt.Println("<<< DONE.")
}
