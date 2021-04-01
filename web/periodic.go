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
	"relay/lib"
	"time"

	"github.com/bfix/gospel/logger"
)

// Periodic tasks for service/data maintenance
func periodicTasks(ctx context.Context, epoch int, balancer chan int64) {
	t := time.Now().Unix()

	// check expired transactions (every 15 mins)
	if epoch%(900/cfg.Service.Epoch) == 0 {
		addrIds, err := db.CloseExpiredTransactions(t)
		if err != nil {
			logger.Println(logger.ERROR, "periodic(tx): "+err.Error())
		} else {
			// check balance of all effected addresses
			go func() {
				for _, id := range addrIds {
					balancer <- id
				}
			}()
		}
	}
	// update market data (every 6 hrs)
	if epoch%(21600/cfg.Service.Epoch) == 1 {
		// get new exchange rates
		rates, err := lib.GetMarketData(cfg.Market.Fiat, coins, cfg.Market.APIKey)
		if err != nil {
			logger.Println(logger.ERROR, "periodic(market): "+err.Error())
		} else {
			// update rates in coin table
			for coin, rate := range rates {
				if err := db.UpdateRate(coin, rate); err != nil {
					logger.Println(logger.ERROR, "periodic(market): "+err.Error())
				}
			}
		}
	}
	// check balances of address if it is not closed and the last check
	// is older than 6 hrs
	addrIds, err := db.PendingAddresses(cfg.Balancer.Rescan)
	if err != nil {
		logger.Println(logger.ERROR, "rescan: "+err.Error())
	} else {
		// check balance of all effected addresses
		go func() {
			for _, id := range addrIds {
				balancer <- id
			}
		}()
	}
	// check for log rotation
	if epoch%cfg.Service.LogRotate == 0 {
		logger.Rotate()
	}
}
