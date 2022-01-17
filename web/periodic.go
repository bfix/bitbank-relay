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
	txList, err := mdl.GetExpiredTransactions()
	if err != nil {
		logger.Println(logger.ERROR, "[periodic] GetExpiredTxs: "+err.Error())
	} else if len(txList) > 0 {
		logger.Println(logger.INFO, "[periodic] Closing expired transactions...")
		// build unique list of addresses from expired transaction
		list := make(map[int64]bool)
		for txID, addrID := range txList {
			logger.Printf(logger.INFO, "[periodic] Closing transaction #%d", txID)
			if err = mdl.CloseTransaction(txID); err != nil {
				logger.Println(logger.ERROR, "[periodic] CloseTx: "+err.Error())
				continue
			}
			list[addrID] = true
		}
		addrIds := make([]int64, 0)
		for addrID := range list {
			addrIds = append(addrIds, addrID)
		}
		logger.Printf(logger.DBG, "[periodic] => %d addresses effected", len(addrIds))
		// check balance of all effected addresses
		go func() {
			for _, id := range addrIds {
				balancer <- id
			}
		}()
	}
	// update market data
	if epoch%cfg.Market.Rescan == 1 {
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
				if err := mdl.UpdateRate(coin, rate); err != nil {
					logger.Println(logger.ERROR, "[periodic] UpdateRate: "+err.Error())
				}
			}
		}
	}
	// check balances of addresses that need a rescan (balance sync)
	addrIds, err := mdl.PendingAddresses()
	if err != nil {
		logger.Println(logger.ERROR, "[periodic] rescan: "+err.Error())
	} else if len(addrIds) > 0 {
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
