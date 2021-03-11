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
	"log"

	"github.com/bfix/gospel/bitcoin/wallet"
)

func main() {

	// read configuration
	cfg, err := lib.ReadConfig("config.json")
	if err != nil {
		log.Fatal(err)
	}

	// verify handlers
	for _, coin := range cfg.Coins {
		log.Println("--------------------------------")
		log.Printf("Coin: '%s'", coin.Name)

		// construct public HD wallet for account
		pk, err := wallet.ParseExtendedPublicKey(coin.Pk)
		if err != nil {
			log.Printf("Pk: %s\n", err.Error())
			continue
		}
		hd := wallet.NewHDPublic(pk, coin.Path)

		// get base extended public key for given account
		path := coin.Path + "/0/0"
		bpk, err := hd.Public(path)
		if err != nil {
			log.Printf("hd.Public: %s\n", err.Error())
			continue
		}

		// get handler
		hdlr, err := lib.GetHandler(coin.Name)
		if err != nil {
			log.Printf("GetHandler: %s\n", err.Error())
			continue
		}
		// verify handler
		addr, err := hdlr.GetAddress(bpk.Data)
		if err != nil {
			log.Printf("GetAddress: %s\n", err.Error())
			continue
		}
		if addr != coin.Addr {
			log.Printf("addr mismatch: %s != %s\n", addr, coin.Addr)
			continue
		}
		log.Println("    * Handler verified")
	}
}
