package main

//----------------------------------------------------------------------
// This file is part of 'bitbank'.
// Copyright (C) 2021 Bernd Fix  >Y<
//
// 'bitbank' is free software: you can redistribute it and/or modify it
// under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// 'bitbank' is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
//
// SPDX-License-Identifier: AGPL3.0-or-later
//----------------------------------------------------------------------

import (
	"addresser/lib"
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/bfix/gospel/bitcoin/wallet"
)

func main() {

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
	cfg, err := lib.ReadConfig("config-template.json")
	if err != nil {
		fmt.Println("<<< ERROR: " + err.Error())
		return
	}
	// process all entries
	for _, coin := range cfg.Coins {
		fmt.Printf("<<<    Processing '%s'...\n", coin.Name)
		version := coin.GetXDVersion()
		if version == 0 {
			fmt.Printf("<<< ERROR: No valid version specified (%s)\n", coin.Name)
			continue
		}
		// get base extended public key for given account
		bpk, err := hd.Public(coin.Path)
		if err != nil {
			fmt.Println("<<< ERROR: " + err.Error())
			continue
		}
		bpk.Data.Version = version
		coin.Pk = bpk.String()

		// get handler
		hdlr, err := lib.GetHandler(coin.Name)
		if err != nil {
			fmt.Println("<<< ERROR: " + err.Error())
			continue
		}
		// compute base account address
		bpk, err = hd.Public(coin.Path + "/0/0")
		if err != nil {
			fmt.Println("<<< ERROR: " + err.Error())
			continue
		}
		bpk.Data.Version = version
		addr, err := hdlr.GetAddress(bpk.Data)
		if err != nil {
			fmt.Println("<<< ERROR: " + err.Error())
			continue
		}
		coin.Addr = addr
	}
	// save to configuration file
	if err = lib.WriteConfig("config.json", cfg); err != nil {
		fmt.Println("<<< ERROR: " + err.Error())
		return
	}
	fmt.Println("<<< DONE.")
}
