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

	"github.com/bfix/gospel/logger"
)

// RateLimiter computes rate limit-compliant delays for requests
type RateLimiter struct {
	rates []int      // rates [sec, min, hr, day, week]
	lock  sync.Mutex // one request at a time
	last  *entry     // reference to last entry
}

// NewRateLimiter creates a newly initialitzed rate limiter.
func NewRateLimiter(rate ...int) *RateLimiter {
	lim := new(RateLimiter)
	lim.rates = rate
	lim.last = newEntry()
	return lim
}

// Pass waits for a rate limit-compliant delay before passing a new request
func (lim *RateLimiter) Pass() {
	// only one request at a time
	lim.lock.Lock()
	defer lim.lock.Unlock()

	// get current timestamp
	ts := time.Now().Unix()

	// calculate rates
	pSec, pMin, pHr, pDay, pWeek := 0, 0, 0, 0, 0
	var e, next, xHr, xDay, xWeek *entry
loop:
	for e, next = lim.last, nil; e != nil; next, e = e, e.prev {
		tDiff := ts - e.ts
		switch {
		case tDiff > 3600*24*7:
			// drop out-of-range entries (one week)
			next.prev = nil
			for e != nil {
				e = e.drop()
			}
			// we are done
			break loop
		// Count events in time-slot
		case tDiff > 3600*24:
			pWeek++
		case tDiff > 3600:
			xWeek = e
			pDay++
		case tDiff > 60:
			xDay = e
			pHr++
		case tDiff > 0:
			xHr = e
			pMin++
		case tDiff == 0:
			pSec++
		}
	}
	pMin += pSec
	pHr += pMin
	pDay += pHr
	pWeek += pDay

	// check compliance with given rates; compute wait time
	// for next accepted request
	num := len(lim.rates)
	var delay int = 0
	switch {
	case num > 0 && pSec+1 > lim.rates[0]:
		delay = 1
	case num > 1 && pMin+1 > lim.rates[1]:
		delay = 60 - int(ts-xHr.ts)
	case num > 2 && pHr+1 > lim.rates[2]:
		delay = 3600 - int(ts-xDay.ts)
	case num > 3 && pDay+1 > lim.rates[3]:
		delay = 24*3600 - int(ts-xWeek.ts)
	case num > 4 && pWeek+1 > lim.rates[4]:
		delay = 7*24*3600 - int(ts-xWeek.ts)
	}
	// delay for given time
	if delay > 0 {
		logger.Printf(logger.DBG, "RateLimit: Delaying for %d seconds", delay)
		time.Sleep(time.Duration(delay) * time.Second)
	}
	// prepend new request at beginning of list
	e = newEntry()
	e.prev = lim.last
	lim.last = e
}

//----------------------------------------------------------------------
// Helper types
//----------------------------------------------------------------------

// Entry in a single-linked list
type entry struct {
	ts   int64
	prev *entry
}

// Drop the entry (cut link)
// Returns the linked entry.
func (e *entry) drop() *entry {
	p := e.prev
	e.prev = nil
	return p
}

// Return a new request entry
func newEntry() *entry {
	return &entry{
		ts:   time.Now().Unix(),
		prev: nil,
	}
}
