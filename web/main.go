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

var (
	handlers = make(map[string]*lib.Handler)
)

func main() {

	// read configuration
	log.Println("Reading configuration...")
	cfg, err := lib.ReadConfig("config.json")
	if err != nil {
		log.Fatal(err)
	}

	// load handlers
	log.Println("Initializing coin handlers:")
	for _, coin := range cfg.Coins {
		log.Printf("   * %s (%s)", coin.Name, coin.Descr)

		// get coin handler
		hdlr, err := lib.NewHandler(coin, wallet.AddrMain)
		if err != nil {
			fmt.Println("<<< ERROR: " + err.Error())
			continue
		}
		// verify handler
		addr, err := hdlr.GetAddress(0)
		if err != nil {
			log.Println("<<< ERROR: " + err.Error())
			continue
		}
		if addr != coin.Addr {
			log.Printf("<<< ERROR: %s != %s\n", addr, coin.Addr)
			continue
		}
		// save handler
		handlers[coin.Name] = hdlr
	}
	log.Println("Done.")

	// setting up webservice

}
