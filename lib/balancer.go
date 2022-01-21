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

package lib

import (
	"context"
	"fmt"

	"github.com/bfix/gospel/logger"
)

// Error codes
var (
	ErrBalanceFailed       = fmt.Errorf("balance query failed")
	ErrBalanceAccessDenied = fmt.Errorf("HTTP GET access denied")
)

// StartBalancer starts the background balance processor.
// It returns a channel for balance check requests that accepts int64
// values that refer to the model id of the address record
// that is to be checked.
func StartBalancer(ctx context.Context, mdl *Model) chan int64 {
	// start background process
	ch := make(chan int64)
	running := make(map[int64]bool)
	pid := 0
	go func() {
		for {
			select {
			// handle balance requests
			case ID := <-ch:
				// close processor on negative row id
				if ID < 0 {
					close(ch)
					return
				}
				// ignore request for already pending address
				if _, ok := running[ID]; ok {
					break
				}
				running[ID] = true

				// get address information
				addr, coin, balance, rate, err := mdl.GetAddressInfo(ID)
				if err != nil {
					logger.Printf(logger.ERROR, "Balancer: can't retrieve address #%d", ID)
					logger.Println(logger.ERROR, "=> "+err.Error())
					break
				}
				pid++
				logger.Printf(logger.INFO, "Balancer[%d] update addr=%s (%f %s)...", pid, addr, balance, coin)

				// get new address balance
				go func(pid int) {
					flag := false
					defer func() {
						mdl.NextUpdate(ID, flag)
						delete(running, ID)
					}()
					// get matching handler
					hdlr, ok := HdlrList[coin]
					if !ok {
						logger.Printf(logger.ERROR, "Balancer[%d] No handler for '%s'", pid, coin)
						return
					}
					// perform balance check
					newBalance, err := hdlr.GetBalance(addr)
					if err != nil {
						logger.Printf(logger.ERROR, "Balancer[%d] sync failed: %s", pid, err.Error())
						return
					}
					// update balance if increased
					diff := newBalance - balance
					if diff < 1e-8 {
						logger.Printf(logger.INFO, "Balancer[%d] unchanged balance (%f)", pid, balance)
						return
					}
					logger.Printf(logger.INFO, "Balancer[%d] => new balance: %f", pid, newBalance)
					flag = true

					// update balance in model
					if err = mdl.UpdateBalance(ID, newBalance); err != nil {
						logger.Printf(logger.ERROR, "Balancer[%d] update failed: %s", pid, err.Error())
						return
					}
					// record incoming funds
					if err = mdl.Incoming(ID, diff); err != nil {
						logger.Printf(logger.ERROR, "Balancer[%d] record incoming failed: %s", pid, err.Error())
						return
					}
					// check if account limit is reached...
					if hdlr.limit < balance*rate {
						// yes: close address
						logger.Printf(logger.INFO, "Balancer[%d]: Closing address '%s' with balance=%f", pid, addr, balance)
						if err = mdl.CloseAddress(ID); err != nil {
							logger.Printf(logger.ERROR, "Balancer[%d] CloseAddress: %s", pid, err.Error())
							return
						}
					}
				}(pid)

			// cancel processor
			case <-ctx.Done():
				close(ch)
				return
			}
		}
	}()
	return ch
}
