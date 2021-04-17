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
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {

	print := func(stats *RateStats) {
		t.Log("-----------------------------------")
		t.Log(" Second Minute   Hour    Day   Week")
		t.Logf("%7d%7d%7d%7d%7d\n", stats.pSec, stats.pMin, stats.pHr, stats.pDay, stats.pWeek)
		t.Logf("%7d%7d%7d%7d%7d\n", stats.rSec, stats.rMin, stats.rHr, stats.rDay, stats.rWeek)
		t.Log("-----------------------------------")
	}

	// lim := NewRateLimiter(5, 150, 450, 5000, 20000)
	lim := NewRateLimiter(0, 30, 0, 1440)
	for i := 0; i < 200; i++ {
		time.Sleep(time.Second)
		stats := lim.Stats()
		print(stats)
		delay := stats.Wait()
		t.Logf("Delaying %d seconds...\n", delay)
		time.Sleep(time.Duration(delay) * time.Second)
		e := newEntry()
		e.prev = lim.last
		lim.last = e
	}
}
