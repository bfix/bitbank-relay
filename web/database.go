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
	"database/sql"
	"relay/lib"

	"github.com/bfix/gospel/bitcoin/wallet"
	_ "github.com/go-sql-driver/mysql"
)

// Database for persistent storage
type Database struct {
	inst *sql.DB
}

// Connect to database
func Connect(cfg *lib.DatabaseConfig) (db *Database, err error) {
	db = &Database{}
	db.inst, err = sql.Open(cfg.Mode, cfg.Connect)
	return
}

// GetCoin get the database identifier for a goven coin.
// An entry is created if the coin is not found.
func (db *Database) GetCoin(symb string) (id int64, isNew bool, err error) {
	row := db.inst.QueryRow("select id from coin where symbol = ?", symb)
	if err = row.Scan(&id); err != nil {
		var res sql.Result
		ref, name := wallet.GetCoinInfo(symb)
		if res, err = db.inst.Exec("insert into coin(symbol,descr,ref) values(?,?,?)", symb, name, ref); err != nil {
			return
		}
		id, err = res.LastInsertId()
		isNew = true
		return
	}
	return
}

// GetCurrentAddress returns the currently used address for a given coin.
func (db *Database) GetCurrentAddress(coinID int64) (addr string, idx int, err error) {
	row := db.inst.QueryRow("select val,idx from addr where active and coin=?", coinID)
	err = row.Scan(&addr, &idx)
	return
}

// ReplaceAddress replaces an old address with a new one
func (db *Database) ReplaceAddress(coinID int64, oldAddr, newAddr string, idx int) (err error) {
	_, err = db.inst.Exec("update addr set active=0, lastSeen=now() where coin=? and val=?", coinID, oldAddr)
	if err != nil {
		return
	}
	_, err = db.inst.Exec("insert into addr(coin,val,idx) values(?,?,?)", coinID, newAddr, idx)
	return
}

// Close database connection
func (db *Database) Close() error {
	return db.inst.Close()
}
