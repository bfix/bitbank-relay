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
	"context"
	"time"

	"github.com/bfix/gospel/logger"
)

func runService(ctx context.Context) error {

	logger.Println(logger.INFO, "Starting web service...")
	_, cancel := context.WithCancel(ctx)
	go func() {
		time.Sleep(2 * time.Minute)
		logger.Println(logger.DBG, "Cancelling service")
		cancel()
	}()

	logger.Println(logger.INFO, "Waiting for client requests...")
	return nil
}

func periodicTasks(ctx context.Context) {

}
