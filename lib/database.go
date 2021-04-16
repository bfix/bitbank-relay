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

	// import MySQL driver
	_ "github.com/go-sql-driver/mysql"
	// import SQLite3 driver
	_ "github.com/mattn/go-sqlite3"
)

// Error codes
var (
	ErrDatabaseNotAvailable = fmt.Errorf("Database not available")
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
func (db *Database) Close() (err error) {
	if db.inst != nil {
		err = db.inst.Close()
	}
	return
}

//----------------------------------------------------------------------
// Coin-related methods
//----------------------------------------------------------------------

// CoinInfo contains information about a coin
type CoinInfo struct {
	Symbol string  `json:"symb"`
	Label  string  `json:"label"`
	Logo   string  `json:"logo"`
	Rate   float64 `json:"rate"` // price of coin in fiat currency
}

// AccCoinInfo holds information about a coin and the
// accumulated balance of the coin over all accounts.
type AccCoinInfo struct {
	CoinInfo
	Total float64 `json:"total"` // total balance in coins
}

// GetCoins returns a list of coins for a give account
func (db *Database) GetCoins(account string) ([]*CoinInfo, error) {
	// check for valid database
	if db.inst == nil {
		return nil, ErrDatabaseNotAvailable
	}
	// select coins for given account
	rows, err := db.inst.Query("select coin,label,logo,rate from coins4account where account=?", account)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*CoinInfo, 0)
	for rows.Next() {
		e := new(CoinInfo)
		if err = rows.Scan(&e.Symbol, &e.Label, &e.Logo, &e.Rate); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, nil
}

// GetCoin get information for a given coin.
func (db *Database) GetCoin(symb string) (ci *CoinInfo, err error) {
	// check for valid database
	if db.inst == nil {
		return nil, ErrDatabaseNotAvailable
	}
	// select coin information
	row := db.inst.QueryRow("select label,logo,rate from coin where symbol=?", symb)
	ci = new(CoinInfo)
	ci.Symbol = symb
	err = row.Scan(&ci.Label, &ci.Logo, &ci.Rate)
	return
}

// GetAccumulatedCoins returns information about a coin and its accumulated
// balance over all accounts.
func (db *Database) GetAccumulatedCoins() (aci []*AccCoinInfo, err error) {
	// check for valid database
	if db.inst == nil {
		return nil, ErrDatabaseNotAvailable
	}
	// select coin information
	query := `
		select
			c.symbol as symbol,
			c.label as label,
			c.logo as logo,
			c.rate as rate,
			sum(a.balance) as total
		from
			coin c, addr a
		where
			c.id = a.coin
		group by c.id
	`
	var rows *sql.Rows
	if rows, err = db.inst.Query(query); err != nil {
		return
	}
	for rows.Next() {
		ci := new(AccCoinInfo)
		if err = rows.Scan(&ci.Symbol, &ci.Label, &ci.Logo, &ci.Rate, &ci.Total); err != nil {
			return
		}
		aci = append(aci, ci)
	}
	return
}

// SetCoinLogo sets a base64-encoded SVG logo for a coin
func (db *Database) SetCoinLogo(coin, logo string) error {
	// check for valid database
	if db.inst == nil {
		return ErrDatabaseNotAvailable
	}
	// set new coin logo in database
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
	// check for valid database
	if db.inst == nil {
		return "", ErrDatabaseNotAvailable
	}
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

// PendingAddresses returns a list of open addresses with a balance check
// older than 't' seconds.
func (db *Database) PendingAddresses(t int64) ([]int64, error) {
	// check for valid database
	if db.inst == nil {
		return nil, ErrDatabaseNotAvailable
	}
	// get list of pending addresses
	now := time.Now().Unix()
	rows, err := db.inst.Query("select id from addr where stat=0 and (?-lastCheck)>?", now, t)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := make([]int64, 0)
	var ID int64
	for rows.Next() {
		if err = rows.Scan(&ID); err != nil {
			return nil, err
		}
		res = append(res, ID)
	}
	return res, nil
}

// CloseAddress locks an address; no further usage (except spending)
func (db *Database) CloseAddress(ID int64) error {
	// check for valid database
	if db.inst == nil {
		return ErrDatabaseNotAvailable
	}
	// close address in database
	_, err := db.inst.Exec("update addr set stat=1, validTo=now() where id=?", ID)
	return err
}

// GetAddressInfo returns basic info about an address
func (db *Database) GetAddressInfo(ID int64) (addr, coin string, balance, rate float64, err error) {
	// check for valid database
	if db.inst == nil {
		return "", "", 0, 0, ErrDatabaseNotAvailable
	}
	// get information about coin address
	row := db.inst.QueryRow("select coin,val,balance,rate from v_addr where id=?", ID)
	err = row.Scan(&coin, &addr, &balance, &rate)
	return
}

// UpdateBalance sets the new balance for an address
func (db *Database) UpdateBalance(ID int64, balance float64) error {
	// check for valid database
	if db.inst == nil {
		return ErrDatabaseNotAvailable
	}
	// update balance in database
	_, err := db.inst.Exec(
		"update addr set balance=?, lastCheck=? where id=?",
		balance, time.Now().Unix(), ID)
	return err
}

//----------------------------------------------------------------------
// Account-related methods
//----------------------------------------------------------------------

// AccntInfo holds information about an account in the database.
type AccntInfo struct {
	Label string  // account label
	Name  string  // account name
	Total float64 // total balance of account (in fiat currency)
}

// GetAccounts list all accounts with their total balance (in fiat currency)
func (db *Database) GetAccounts() (accnts []*AccntInfo, err error) {
	// check for valid database
	if db.inst == nil {
		return nil, ErrDatabaseNotAvailable
	}
	// select account information
	query := `
		select
			a.label as label,
			a.name as name,
			sum(b.balance*c.rate) as total
		from
			account a, addr b, coin c
		where
			a.id = b.accnt and
			c.id = b.coin
		group by a.id
	`
	var rows *sql.Rows
	if rows, err = db.inst.Query(query); err != nil {
		return
	}
	for rows.Next() {
		ai := new(AccntInfo)
		if err = rows.Scan(&ai.Label, &ai.Name, &ai.Total); err != nil {
			return
		}
		accnts = append(accnts, ai)
	}
	return
}

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
	// check for valid database
	if db.inst == nil {
		return nil, ErrDatabaseNotAvailable
	}
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
		ValidTo:   now + 900,
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
	// check for valid database
	if db.inst == nil {
		return nil, ErrDatabaseNotAvailable
	}
	// get information about transaction from database
	tx = new(Transaction)
	tx.ID = txid
	row := db.inst.QueryRow(
		"select addr,coin,account,stat,validFrom,validTo from v_tx where txid=?", txid)
	err = row.Scan(&tx.Addr, &tx.Coin, &tx.Accnt, &tx.Status, &tx.ValidFrom, &tx.ValidTo)
	return
}

// GetExpiredTransactions collects transactions that have expired.
// Returns a mapping between transaction and associated address.
func (db *Database) GetExpiredTransactions() (map[int64]int64, error) {
	// check for valid database
	if db.inst == nil {
		return nil, ErrDatabaseNotAvailable
	}
	// collect expired transactions
	t := time.Now().Unix()
	rows, err := db.inst.Query("select id,addr from tx where stat=0 and validTo<?", t)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make(map[int64]int64)
	for rows.Next() {
		// get identifiers for tx and address
		var txID, addrID int64
		if err = rows.Scan(&txID, &addrID); err != nil {
			return nil, err
		}
		list[txID] = addrID
	}
	return list, nil
}

// CloseTransaction closes a pending transaction.
func (db *Database) CloseTransaction(txID int64) error {
	// check for valid database
	if db.inst == nil {
		return ErrDatabaseNotAvailable
	}
	// close transaction in database
	_, err := db.inst.Exec("update tx set stat=1 where id=?", txID)
	return err
}

//----------------------------------------------------------------------
// Market-related methods
//----------------------------------------------------------------------

// UpdateRate sets the new exchange rate (in market base currency) for
// the given coin.
func (db *Database) UpdateRate(coin string, rate float64) error {
	// check for valid database
	if db.inst == nil {
		return ErrDatabaseNotAvailable
	}
	// update rate in coin record
	_, err := db.inst.Exec("update coin set rate=? where symbol=?", rate, coin)
	return err
}
