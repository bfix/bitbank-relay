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

	"github.com/bfix/gospel/logger"
)

// Periodic tasks for service/data maintenance
func periodicTasks(ctx context.Context, epoch int, balancer chan int64) {

	// check expired transactions
	logger.Println(logger.INFO, "[periodic] Closing expired transactions...")
	addrIds, err := db.CloseExpiredTransactions()
	if err != nil {
		logger.Println(logger.ERROR, "[periodic] CloseExpiredTxs: "+err.Error())
	} else if len(addrIds) > 0 {
		logger.Printf(logger.DBG, "[periodic] => %d addresses effected", len(addrIds))
		// check balance of all effected addresses
		go func() {
			for _, id := range addrIds {
				balancer <- id
			}
		}()
	}
	// update market data (every 6 hrs)
	if epoch%(21600/cfg.Service.Epoch) == 1 {
		// get new exchange rates
		logger.Println(logger.INFO, "[periodic] Get market data...")
		rates, err := lib.GetMarketData(cfg.Market.Fiat, coins, cfg.Market.APIKey)
		if err != nil {
			logger.Println(logger.ERROR, "[periodic] GetMarketData: "+err.Error())
		} else {
			logger.Printf(logger.INFO, "[periodic] Updating market data (%d entries)", len(rates))
			// update rates in coin table
			for coin, rate := range rates {
				logger.Printf(logger.DBG, "[periodic]    * %s: %f", coin, rate)
				if err := db.UpdateRate(coin, rate); err != nil {
					logger.Println(logger.ERROR, "[periodic] UpdateRate: "+err.Error())
				}
			}
		}
	}
	// check balances of address if it is not closed and the last check
	// is older than 6 hrs
	if addrIds, err = db.PendingAddresses(cfg.Balancer.Rescan); err != nil {
		logger.Println(logger.ERROR, "[periodic] rescan: "+err.Error())
	} else {
		logger.Printf(logger.INFO, "[periodic] Update %d pending address balances...", len(addrIds))
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
