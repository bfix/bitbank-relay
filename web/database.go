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
	"database/sql"

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
func (db *Database) GetCoin(symb, descr string) (id int64, isNew bool, err error) {
	row := db.inst.QueryRow("select id from coin where symbol = ?", symb)
	if err = row.Scan(&id); err != nil {
		var res sql.Result
		ref := wallet.GetCoinID(symb)
		if res, err = db.inst.Exec("insert into coin(symbol,descr,ref) values(?,?,?)", symb, descr, ref); err != nil {
			return
		}
		id, err = res.LastInsertId()
		isNew = true
		return
	}
	return
}

// Close database connection
func (db *Database) Close() error {
	return db.inst.Close()
}
