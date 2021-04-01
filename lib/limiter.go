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
	"sync"
	"time"
)

type Limiter struct {
	base   int64
	events []int64
	rates  []int
	lock   sync.Mutex
}

func NewLimiter(rate ...int) *Limiter {
	lim := new(Limiter)
	lim.base = time.Now().Unix()
	lim.events = make([]int64, 0, 3600*24*7)
	lim.rates = rate
	return lim
}

func (lim *Limiter) Pass() {
	// only one at a time
	lim.lock.Lock()
	defer lim.lock.Unlock()

	// get current timestamp
	ts := time.Now().Unix()
	// calculate rates
	pSec, pMin, pHr, pDay, pWeek := 0, 0, 0, 0, 0
	skip := 0
	for i, t := range lim.events {
		tDiff := ts - t
		switch {
		case tDiff > 3600*24*7:
			skip = i + 1
			continue
		case tDiff > 3600*24:
			pWeek++
		case tDiff > 3600:
			pDay++
		case tDiff > 60:
			pHr++
		case tDiff > 0:
			pMin++
		case tDiff == 0:
			pSec++
		}
		pMin += pSec
		pHr += pMin
		pDay += pHr
		pWeek += pDay
	}
	// drop events older than a week
	if skip > 0 {
		size := len(lim.events)
		copy(lim.events[0:], lim.events[skip:])
		lim.events = lim.events[:size-skip]
	}
	// compute wait time
	if lim.rates[0] == 0 {
		secs := 60/lim.rates[1] + 2
		time.Sleep(time.Duration(secs) * time.Second)
	} else {
		time.Sleep(10 * time.Second)
	}
	lim.events = append(lim.events, time.Now().Unix())
}
