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
	"fmt"
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

		// get handler
		hdlr, err := lib.GetHandler(coin.Name)
		if err != nil {
			log.Printf("GetHandler: %s\n", err.Error())
			continue
		}
		// compute base account address
		bpk, err := wallet.ParseExtendedPublicKey(coin.Pk)
		if err != nil {
			fmt.Println("<<< ERROR: " + err.Error())
			continue
		}
		bpk.Data.Version = coin.GetXDVersion()
		hdlr.Init(coin.Path, bpk)

		// verify handler
		addr, err := hdlr.GetAddress(0)
		if err != nil {
			log.Printf("<<< ERROR: %s\n", err.Error())
			continue
		}
		if addr != coin.Addr {
			log.Printf("<<< ERROR: %s != %s\n", addr, coin.Addr)
			continue
		}
		log.Println("    * Handler verified")
	}
}
