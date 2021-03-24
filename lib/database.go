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

// CoinInfo contains information about a coin
type CoinInfo struct {
	Symbol string   `json:"symb"`
	Label  string   `json:"label"`
	Logo   string   `json:"logo"`
	Market []*Price `json:"prices,omitempty"`
}

func (db *Database) GetCoins(account string) ([]*CoinInfo, error) {
	rows, err := db.inst.Query("select coin,label,logo from coins4account where account=?", account)
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

// GetCoin get information for a given coin.
func (db *Database) GetCoin(symb string) (ci *CoinInfo, err error) {
	row := db.inst.QueryRow("select label,logo from coin where symbol=?", symb)
	ci = new(CoinInfo)
	ci.Symbol = symb
	err = row.Scan(&ci.Label, &ci.Logo)
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

// Error codes (coin-related)
var (
	ErrDbUnknownCoin = fmt.Errorf("Unknown coin")
)

// GetUnusedAddress returns a currently unused address for a given
// coin/account pair. Creates a new address if none is available.
// (Internal use for generating new transactions)
func (db *Database) getUnusedAddress(dbtx *sql.Tx, coin, account string) (addr string, err error) {

	// do we have a unused address for given coin? if so, use that address.
	row := dbtx.QueryRow(
		"select val from v_addr where stat=0 and coin=? and account=?",
		coin, account)
	err = row.Scan(&addr)
	if err == nil || err != sql.ErrNoRows {
		return
	}
	//  no old address found: generate a new one
	hdlr, ok := HdlrList[coin]
	if !ok {
		err = ErrDbUnknownCoin
		return
	}
	// get coin database id
	var coinID int64
	row = dbtx.QueryRow("select id from coin where symbol=?", coin)
	err = row.Scan(&coinID)
	if err != nil {
		return
	}
	// get account database id
	var accntID int64
	row = dbtx.QueryRow("select id from account where label=?", account)
	err = row.Scan(&accntID)
	if err != nil {
		return
	}
	// get next address index
	var idxV sql.NullInt64
	row = dbtx.QueryRow("select max(idx)+1 from addr where coin=?", coinID)
	if err = row.Scan(&idxV); err != nil {
		return
	}
	idx := int(idxV.Int64)
	if !idxV.Valid {
		idx = 0
	}
	// create and store new address
	if addr, err = hdlr.GetAddress(idx); err != nil {
		return
	}
	_, err = dbtx.Exec("insert into addr(coin,accnt,idx,val) values(?,?,?,?)", coinID, accntID, idx, addr)
	return
}

//----------------------------------------------------------------------
// Account-related methods
//----------------------------------------------------------------------

//----------------------------------------------------------------------
// Transaction-related methods
//----------------------------------------------------------------------

// Transaction is a pending/closed coin transaction
type Transaction struct {
	ID        string `json:"id"`
	Addr      string `json:"addr"`
	Accnt     string `json:"account"`
	Coin      string `json:"coin"`
	Status    int    `json:"status"`
	ValidFrom int64  `json:"validFrom"`
	ValidTo   int64  `json:"validTo"`
}

// NewTransaction creates a new pending transaction for a given coin/account pair
func (db *Database) NewTransaction(coin, account string) (tx *Transaction, err error) {

	// start database transaction
	ctx := context.Background()
	var dbtx *sql.Tx
	if dbtx, err = db.inst.BeginTx(ctx, nil); err != nil {
		return
	}
	// get an address
	var addr string
	if addr, err = db.getUnusedAddress(dbtx, coin, account); err != nil {
		dbtx.Rollback()
		return
	}

	// initialize values
	now := time.Now().Unix()
	idData := make([]byte, 32)
	rand.Read(idData)

	// assemble transaction
	tx = &Transaction{
		ID:        hex.EncodeToString(idData),
		Addr:      addr,
		Status:    0,
		ValidFrom: now,
		ValidTo:   now + 7200,
	}
	var addrID int64
	var accnt sql.NullString
	row := dbtx.QueryRow("select id,coin,account from v_addr where val=?", addr)
	if err = row.Scan(&addrID, &tx.Coin, &accnt); err != nil {
		dbtx.Rollback()
		return
	}
	if accnt.Valid {
		tx.Accnt = accnt.String
	}
	// insert transaction into database
	if _, err = dbtx.Exec(
		"insert into tx(txid,addr,validFrom,validTo) values(?,?,?,?)",
		tx.ID, addrID, tx.ValidFrom, tx.ValidTo); err != nil {
		dbtx.Rollback()
		return
	}
	// increment ref counter in address
	if _, err = dbtx.Exec("update addr set refCnt = refCnt + 1 where id=?", addrID); err != nil {
		dbtx.Rollback()
		return
	}
	// commit database transaction
	err = dbtx.Commit()
	return
}

// GetTransaction returns the Tx instance for a given identifier
func (db *Database) GetTransaction(txid string) (tx *Transaction, err error) {
	tx = new(Transaction)
	tx.ID = txid
	row := db.inst.QueryRow(
		"select addr,coin,account,stat,validFrom,validTo from v_tx where txid=?", txid)
	err = row.Scan(&tx.Addr, &tx.Coin, &tx.Accnt, &tx.Status, &tx.ValidFrom, &tx.ValidTo)
	return
}

//----------------------------------------------------------------------
// Market-related methods
//----------------------------------------------------------------------

type Price struct {
}
