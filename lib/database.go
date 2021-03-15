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
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Database for persistent storage
type Database struct {
	inst *sql.DB
}

// Connect to database
func Connect(cfg *DatabaseConfig) (db *Database, err error) {
	db = &Database{}
	db.inst, err = sql.Open(cfg.Mode, cfg.Connect)
	return
}

// Close database connection
func (db *Database) Close() error {
	return db.inst.Close()
}

//----------------------------------------------------------------------
// Coin-related methods
//----------------------------------------------------------------------

// Error codes (coin-related)
var (
	ErrDbUnknownCoin = fmt.Errorf("Unknown coin")
)

// CoinInfo contains information about a coin
type CoinInfo struct {
	Symbol string   `json:"symb"`
	Label  string   `json:"label"`
	Logo   string   `json:"logo"`
	Market []*Price `json:"prices"`
}

func (db *Database) GetCoins(account string) ([]*CoinInfo, error) {
	rows, err := db.inst.Query("select symb,label,logo from coins4account where ref=?", account)
	if err != nil {
		return nil, err
	}
	list := make([]*CoinInfo, 0)
	for rows.Next() {
		e := new(CoinInfo)
		if err = rows.Scan(&e.Symbol, &e.Label, &e.Logo); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, nil
}

// GetCoin get the database identifier for a given coin.
// An entry is created if the coin is not found.
func (db *Database) GetCoinID(symb string) (id int64, err error) {
	row := db.inst.QueryRow("select id from coin where symbol=?", symb)
	err = row.Scan(&id)
	return
}

// SetCoinLogo sets a base64-encoded SVG logo for a coin
func (db *Database) SetCoinLogo(coin, logo string) error {
	_, err := db.inst.Exec("update coin set logo=? where symbol=?", logo, coin)
	return err
}

//----------------------------------------------------------------------
// Address-related methods
//----------------------------------------------------------------------

// GetUnusedAddress returns a currently unused address for a given coin.
// Creates a new one if none is available.
func (db *Database) GetUnusedAddress(coin string, coinID int64) (addr string, addrID int64, err error) {
	// get coin id if no specified
	if coinID == 0 {
		if coinID, err = db.GetCoinID(coin); err != nil {
			return
		}
	}
	// select (oldest) unused address
	row := db.inst.QueryRow("select id,val from addr where coin=? and stat=0 order by idx asc limit 1", coinID)
	if err = row.Scan(&addrID, &addr); err != nil {
		if err != sql.ErrNoRows {
			return
		}
		// no address found: generate a new one
		hdlr, ok := HdlrList[coin]
		if !ok {
			err = ErrDbUnknownCoin
			return
		}
		// get next address index
		var idx int64
		row = db.inst.QueryRow("select max(idx)+1 from addr where coin=?", coinID)
		if err = row.Scan(&idx); err != nil {
			return
		}
		// create and store new address
		if addr, err = hdlr.GetAddress(int(idx)); err != nil {
			return
		}
		var res sql.Result
		if res, err = db.inst.Exec("insert into addr(coin,idx,val) values(?,?,?)", coinID, idx, addr); err != nil {
			return
		}
		addrID, err = res.LastInsertId()
	}
	return
}

//----------------------------------------------------------------------
// Account-related methods
//----------------------------------------------------------------------

// GetAccount returns the database id of an account reference
func (db *Database) GetAccountID(ref string) (id int64, err error) {
	row := db.inst.QueryRow("select id from account where ref=?", ref)
	err = row.Scan(&id)
	return
}

//----------------------------------------------------------------------
// Transaction-related methods
//----------------------------------------------------------------------

// Transaction is a pending/closed coin transaction
type Transaction struct {
	ID        string `json:"id"`
	Addr      int64  `json:"addr"`
	Accnt     int64  `json:"account"`
	Status    int    `json:"status"`
	ValidFrom int64  `json:"validFrom"`
	ValidTo   int64  `json:"validTo"`
}

// NewTransaction creates a new pending transaction for a given address
func (db *Database) NewTransaction(addrID int64, accountID int64) (tx *Transaction, err error) {
	// initialize values
	now := time.Now().Unix()
	idData := make([]byte, 32)
	rand.Read(idData)

	// assemble transaction
	tx = &Transaction{
		ID:        hex.EncodeToString(idData),
		Addr:      addrID,
		Accnt:     accountID,
		Status:    0,
		ValidFrom: now,
		ValidTo:   now + 7200,
	}
	// insert transaction into database
	_, err = db.inst.Exec(
		"insert into tx(txid,addr,accnt,validFrom,validTo) values(?,?,?,?,?)",
		tx.ID, addrID, accountID, tx.ValidFrom, tx.ValidTo)
	return
}

//----------------------------------------------------------------------
// Market-related methods
//----------------------------------------------------------------------

type Price struct {
}
