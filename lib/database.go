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

	"github.com/bfix/gospel/logger"

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
// Generic item
//----------------------------------------------------------------------

// Item represents either a coin or an account. ID is refering to the record
// in the database, Name is the common name and Status indicates if a condition
// for the item is statisfied. A coin condition is "assigned to account" and
// an account condition is "assigned to a coin"
type Item struct {
	ID     int64
	Name   string
	Status bool
}

// Get a list of items from a query; the query must return three columns
// corresponding with the fields of the Item struct.
func (db *Database) getItems(query string, id int64) (list []*Item, err error) {
	var rows *sql.Rows
	if rows, err = db.inst.Query(query, id); err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		item := new(Item)
		if err = rows.Scan(&item.ID, &item.Name, &item.Status); err != nil {
			return
		}
		list = append(list, item)
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
	ID     int64   `json:"id"`     // database ID of coin entry
	Total  float64 `json:"total"`  // total balance in coins
	Accnts []*Item `json:"accnts"` // (assigned) accounts
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
// balance over all accounts. If "coin" is "0", all coins are returned.
func (db *Database) GetAccumulatedCoins(coin int64) (aci []*AccCoinInfo, err error) {
	// check for valid database
	if db.inst == nil {
		return nil, ErrDatabaseNotAvailable
	}
	// select coin information
	query := `
		select
			c.id as id,
			c.symbol as symbol,
			c.label as label,
			c.logo as logo,
			c.rate as rate,
			sum(a.balance) as total
		from
			coin c, addr a
		where
			c.id = a.coin`
	if coin != 0 {
		query += fmt.Sprintf(" and c.id=%d", coin)
	}
	query += " group by c.id"

	var rows *sql.Rows
	if rows, err = db.inst.Query(query); err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		ci := new(AccCoinInfo)
		if err = rows.Scan(&ci.ID, &ci.Symbol, &ci.Label, &ci.Logo, &ci.Rate, &ci.Total); err != nil {
			return
		}
		if ci.Accnts, err = db.getItems(
			"select id,name,(id in (select accnt from accept where coin=?)) as used from account",
			ci.ID); err != nil {
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

// AddrInfo holds information about an address
type AddrInfo struct {
	ID         int64   `json:"id"`         // database id of address entry
	Status     int     `json:"status"`     // address status
	Coin       string  `json:"coin"`       // name of coin
	Account    string  `json:"account"`    // name of account
	Val        string  `json:"value"`      // address value
	Balance    float64 `json:"balance"`    // address balance
	Rate       float64 `json:"rate"`       // coin value (price per coin)
	RefCount   int     `json:"refCount"`   // number of transactions
	LastCheck  string  `json:"lastCheck"`  // last balance check
	ValidSince string  `json:"validSince"` // start of active period
	ValidUntil string  `json:"validUntil"` // end of active period
}

// GetAddress returns a list of active adresses
func (db *Database) GetAddresses() (ai []*AddrInfo, err error) {
	// check for valid database
	if db.inst == nil {
		return nil, ErrDatabaseNotAvailable
	}
	// get information about active addresses
	var rows *sql.Rows
	query := `
		select id,coin,val,balance,rate,stat,account,cnt,lastCheck,validFrom,validTo
		from v_addr
		where stat < 2
		order by balance*rate desc
	`
	if rows, err = db.inst.Query(query); err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		addr := new(AddrInfo)
		var (
			last     sql.NullInt64
			from, to sql.NullString
		)
		if err = rows.Scan(
			&addr.ID, &addr.Coin, &addr.Val, &addr.Balance, &addr.Rate, &addr.Status,
			&addr.Account, &addr.RefCount, &last, &from, &to); err != nil {
			return
		}
		if last.Valid {
			addr.LastCheck = time.Unix(last.Int64, 0).Format(time.RFC1123)
		}
		if from.Valid {
			addr.ValidSince = from.String
		}
		if to.Valid {
			addr.ValidUntil = to.String
		}
		ai = append(ai, addr)
	}
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
// Assignement-related methods.
//----------------------------------------------------------------------

// CountAssignments returns the number of assignments between coins and
// accounts. An ID of "0" means "all".
func (db *Database) CountAssignments(coin, accnt int64) int {
	// assemble WHERE clause
	clause := ""
	addClause := func(id int64, field string) {
		if id != 0 {
			if len(clause) > 0 {
				clause += " and"
			}
			clause += fmt.Sprintf(" %s=%d", field, id)
		}
	}
	addClause(coin, "coin")
	addClause(accnt, "accnt")

	// assemble query
	query := "select count(*) from accept"
	if len(clause) > 0 {
		query += " where" + clause
	}
	row := db.inst.QueryRow(query)
	var count int
	if err := row.Scan(&count); err != nil {
		logger.Printf(logger.ERROR, "CountAssign: "+err.Error())
		count = -1
	}
	return count
}

// ChangeAssignment adds or removes coin/account assignments
func (db *Database) ChangeAssignment(coin, accnt int64, add bool) (err error) {
	if add {
		_, err = db.inst.Exec("insert ignore into accept(coin,accnt) values(?,?)", coin, accnt)
	} else {
		_, err = db.inst.Exec("delete from accept where coin=? and accnt=?", coin, accnt)
	}
	return
}

//----------------------------------------------------------------------
// Account-related methods
//----------------------------------------------------------------------

// AccntInfo holds information about an account in the database.
type AccntInfo struct {
	ID    int64   `json:"id"`    // database ID of account record
	Label string  `json:"label"` // account label
	Name  string  `json:"name"`  // account name
	Total float64 `json:"total"` // total balance of account (in fiat currency)
	Coins []*Item `json:"coins"` // (assigned) coins
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
			a.id as id,
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
	defer rows.Close()
	for rows.Next() {
		ai := new(AccntInfo)
		if err = rows.Scan(&ai.ID, &ai.Label, &ai.Name, &ai.Total); err != nil {
			return
		}
		if ai.Coins, err = db.getItems(
			"select id,label,(id in (select coin from accept where accnt=?)) as used from coin",
			ai.ID); err != nil {
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
