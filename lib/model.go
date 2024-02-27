//----------------------------------------------------------------------
// This file is part of 'bitbank-relay'.
// Copyright (C) 2021-2024, Bernd Fix >Y<
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
//
// Abstract persistent data model for all bitbank-relay services.
// The model provides manipulation and query methods that represent
// its logic:
//
// Table 'coin' has all excepted cryptocoins; table 'account' has all
// receivers and table 'accept' maps which coins are accepted by which
// receivers. For each records in 'accept' there is an 'addr' record,
// that corresponds to the currently used receiving address for the
// 'accept' record.
//
// If a client requests an address for a 'account'/'coin' pair, this
// address is returned. If the address is not defined, it is generated
// from a HDKD wallet (by index). The time of the last client delivery
// is recorded in the 'addr' record.
//
// Any existing 'addr' can be in one of three states:
//   (0) The address is in use (for client delivery)
//   (1) The address is closed (will no longer be used)
//   (2) The address is locked (will no longer be updated)
//
// The 'addr' record has some additional information used for automatic
// balance checks. It keeps the time of last balance check, the current
// wait time be tween checks and the time of the next update.
//
// All 'addr' records in state (0) or (1) will have balance updates
// at specified times. Whenever the address is requested by a client,
// the wait time will be set to 300 seconds (5 min) and the next update
// will happen "wait time" from now to check for incoming funds.
//
// If a balance update yields a new balance (higher than before), the
// balance is updated and the new wait time is (re-)set to 300 seconds.
// Otherwise the wait time is doubled but can't exceed a week. Based on
// the wait time a time for the next update is calculated.
//
//----------------------------------------------------------------------

package lib

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/bfix/gospel/logger"

	// import MySQL driver
	_ "github.com/go-sql-driver/mysql"

	// import SQLite3 driver
	_ "github.com/mattn/go-sqlite3"
)

// Error codes
var (
	ErrModelNotAvailable = fmt.Errorf("model not available")
)

// Model for domain logic and persistent storage
type Model struct {
	inst *sql.DB
	cfg  *ModelConfig
}

// Connect to model
func Connect(cfg *ModelConfig) (mdl *Model, err error) {
	mdl = &Model{}
	mdl.cfg = cfg
	mdl.inst, err = sql.Open(cfg.DbEngine, cfg.DbConnect)
	return
}

// Close model connection
func (mdl *Model) Close() (err error) {
	if mdl.inst != nil {
		err = mdl.inst.Close()
	}
	return
}

//----------------------------------------------------------------------
// Generic item
//----------------------------------------------------------------------

// Item represents either a coin or an account. ID is refering to the record
// in the model repository. Name is the common name and Status indicates if
// a condition for the item is statisfied. A coin condition is "assigned to
// account" and an account condition is "assigned to a coin". The item can
// have additional attributes (for display) in the Dictionary field.
type Item struct {
	ID     int64
	Name   string
	Status bool
	Dict   map[string]interface{}
}

// String returns a human-readable item
func (i *Item) String() string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "{ID: %d,", i.ID)
	fmt.Fprintf(buf, "Name: '%s',", i.Name)
	fmt.Fprintf(buf, "Status: %v", i.Status)
	for k, v := range i.Dict {
		fmt.Fprintf(buf, ", %s: '%v'", k, v)
	}
	fmt.Fprintf(buf, "}")
	return buf.String()
}

// Get a list of items from a query; the query must return three columns
// corresponding with the fields of the Item struct. The first three returned
// columns MUST be named "id", "name" and "status" and are assigned to the
// first three fields of the Item; additional coulmns are added to the
// dictionary).
func (mdl *Model) getItems(query string, args ...interface{}) (list []*Item, err error) {
	// perform query
	var rows *sql.Rows
	if rows, err = mdl.inst.Query(query, args...); err != nil {
		return
	}
	defer rows.Close()

	// get returned columns
	var columns []string
	if columns, err = rows.Columns(); err != nil {
		return
	}
	numCols := len(columns)

	// assemble value pointers
	values := make([]interface{}, numCols)
	ptrs := make([]interface{}, numCols)
	for i := range values {
		var stub interface{}
		values[i] = stub
		ptrs[i] = &values[i]
	}

	// parse rows
	for rows.Next() {
		// scan column values
		if err = rows.Scan(ptrs...); err != nil {
			return
		}
		// assemble item
		item := new(Item)
		item.ID = values[0].(int64)
		item.Name = string(values[1].(string))
		item.Status = (values[2].(int64) != 0)
		item.Dict = make(map[string]interface{})
		for i := range values[3:] {
			var val interface{} = nil
			if values[3+i] != nil {
				switch v := values[3+i].(type) {
				case []uint8:
					val = string(v)
				case int32:
					val = int64(v)
				case float32:
					val = float64(v)
				default:
					val = v
				}
			}
			item.Dict[columns[3+i]] = val
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
	ID     int64   `json:"id"`    // repository ID of coin entry
	Symbol string  `json:"symb"`  // Ticker symbol of coin
	Label  string  `json:"label"` // Full coin name
	Logo   string  `json:"logo"`  // SVG-encoded coin logo
	Rate   float64 `json:"rate"`  // price of coin in fiat currency
}

// AccCoinInfo holds information about a coin and the
// accumulated balance of the coin over all accounts.
type AccCoinInfo struct {
	CoinInfo
	Total  float64 `json:"total"`  // total balance in coins
	NumTx  int     `json:"numTx"`  // number of transactions for this coin
	Accnts []*Item `json:"accnts"` // (assigned) accounts
}

// GetCoins returns a list of coins for a given account
func (mdl *Model) GetCoins(account string) ([]*CoinInfo, error) {
	// check for valid repository
	if mdl.inst == nil {
		return nil, ErrModelNotAvailable
	}
	// select coins for given account
	rows, err := mdl.inst.Query("select coinId,coin,label,logo,rate from v_coin_accnt where account=?", account)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*CoinInfo, 0)
	for rows.Next() {
		e := new(CoinInfo)
		if err = rows.Scan(&e.ID, &e.Symbol, &e.Label, &e.Logo, &e.Rate); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, nil
}

// GetCoinInfo returns coin information for given id
func (mdl *Model) GetCoinInfo(coinID int64) (*CoinInfo, error) {
	// check for valid repository
	if mdl.inst == nil {
		return nil, ErrModelNotAvailable
	}
	// select coin for given ID
	row := mdl.inst.QueryRow("select symbol,label,logo,rate from coin where id=?", coinID)
	e := new(CoinInfo)
	e.ID = coinID
	var logo sql.NullString
	err := row.Scan(&e.Symbol, &e.Label, &logo, &e.Rate)
	if logo.Valid {
		e.Logo = logo.String
	}
	return e, err
}

// GetCoin get information for a given coin.
func (mdl *Model) GetCoin(symb string) (ci *CoinInfo, err error) {
	// check for valid repository
	if mdl.inst == nil {
		return nil, ErrModelNotAvailable
	}
	// select coin information
	row := mdl.inst.QueryRow("select id,label,logo,rate from coin where symbol=?", symb)
	ci = new(CoinInfo)
	ci.Symbol = symb
	var logo sql.NullString
	err = row.Scan(&ci.ID, &ci.Label, &logo, &ci.Rate)
	if logo.Valid {
		ci.Logo = logo.String
	}
	return
}

// GetCoinID returns the repository ID of a coin
func (mdl *Model) GetCoinID(label string) (id int64, err error) {
	// check for valid repository
	if mdl.inst == nil {
		return 0, ErrModelNotAvailable
	}
	// query ID
	row := mdl.inst.QueryRow("select id from coin where label=?", label)
	err = row.Scan(&id)
	return
}

// GetAccumulatedCoins returns information about a coin and its accumulated
// balance over all accounts. If "coin" is "0", all coins are returned.
// Locked (state == 2) accounts are not included.
func (mdl *Model) GetAccumulatedCoin(coin int64) (aci []*AccCoinInfo, err error) {
	// check for valid repository
	if mdl.inst == nil {
		return nil, ErrModelNotAvailable
	}
	// select coin information
	query := `
		select
			c.id as id,
			c.symbol as symbol,
			c.label as label,
			c.logo as logo,
			c.rate as rate,
			coalesce(sum(a.balance),0) as total,
			coalesce(sum(a.refCnt),0) as refs
		from coin c
		left join addr a
		on c.id = a.coin and a.stat < 2`
	if coin != 0 {
		query += fmt.Sprintf(" and c.id=%d", coin)
	}
	query += " group by c.id"

	var rows *sql.Rows
	if rows, err = mdl.inst.Query(query); err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		// get basic coin info
		ci := new(AccCoinInfo)
		if err = rows.Scan(&ci.ID, &ci.Symbol, &ci.Label, &ci.Logo, &ci.Rate, &ci.Total, &ci.NumTx); err != nil {
			return
		}
		// get account items
		if ci.Accnts, err = mdl.getItems(`
			select
  				account.id as id,
  				account.name as name,
  				(account.id in (select accnt from accept where coin=?)) as status,
  				sum(addr.balance) as balance,
				count(addr.id) as addrs
			from account
			left join addr on addr.coin=? and addr.stat < 2 and addr.accnt = account.id
			group by account.id`, ci.ID, ci.ID); err != nil {
			return
		}
		// order account by descending balance
		sort.Slice(ci.Accnts, func(i, j int) bool {
			xi := ci.Accnts[i].Dict["balance"]
			bi := -1.
			if xi != nil {
				bi = xi.(float64)
			}
			xj := ci.Accnts[j].Dict["balance"]
			bj := -1.
			if xj != nil {
				bj = xj.(float64)
			}
			return bj < bi
		})
		// logger.Printf(logger.DBG, "Items: %v", ci.Accnts)
		aci = append(aci, ci)
	}
	// sort coins by descending fiat balance
	sort.Slice(aci, func(i, j int) bool {
		return aci[j].Rate*aci[j].Total < aci[i].Rate*aci[i].Total
	})
	return
}

// SetCoinLogo sets a base64-encoded SVG logo for a coin
func (mdl *Model) SetCoinLogo(coin, logo string) error {
	// check for valid repository
	if mdl.inst == nil {
		return ErrModelNotAvailable
	}
	// set new coin logo in model
	_, err := mdl.inst.Exec("update coin set logo=? where symbol=?", logo, coin)
	return err
}

//----------------------------------------------------------------------
// Address-related methods
//----------------------------------------------------------------------

// Error codes (coin-related)
var (
	ErrMdlUnknownCoin = fmt.Errorf("unknown coin")
)

// GetUnusedAddress returns a currently unused address for a given
// coin/account pair. Creates a new address if none is available.
// (Internal use for generating new transactions)
func (mdl *Model) getUnusedAddress(mdltx *sql.Tx, coin, account string) (addr string, err error) {
	// check for valid repository
	if mdl.inst == nil {
		return "", ErrModelNotAvailable
	}
	// do we have a unused address for given coin? if so, use that address.
	row := mdltx.QueryRow(
		"select val from v_addr where stat=0 and coin=? and account=?",
		coin, account)
	err = row.Scan(&addr)
	if err == nil || err != sql.ErrNoRows {
		return
	}
	//  no old address found: generate a new one
	hdlr, ok := HdlrList[coin]
	if !ok {
		err = ErrMdlUnknownCoin
		return
	}
	// get coin id
	var coinID int64
	row = mdltx.QueryRow("select id from coin where symbol=?", coin)
	err = row.Scan(&coinID)
	if err != nil {
		return
	}
	// get account id
	var accntID int64
	row = mdltx.QueryRow("select id from account where label=?", account)
	err = row.Scan(&accntID)
	if err != nil {
		return
	}
	// get next address index
	var idxV sql.NullInt64
	row = mdltx.QueryRow("select max(idx)+1 from addr where coin=?", coinID)
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
	_, err = mdltx.Exec(
		"insert into addr(coin,accnt,idx,val,waitCheck) values(?,?,?,?,?)",
		coinID, accntID, idx, addr, mdl.cfg.BalanceWait[0])
	logger.Printf(logger.INFO, "[addr] New address '%s' for account '%s'", addr, account)
	return
}

// PendingAddresses returns a list of non-locked addresses that are due for
// balance update.
func (mdl *Model) PendingAddresses() ([]int64, error) {
	// check for valid repository
	if mdl.inst == nil {
		return nil, ErrModelNotAvailable
	}
	// get list of pending addresses
	now := time.Now().Unix()
	rows, err := mdl.inst.Query("select id from addr where stat<2 and (?-nextCheck)>=0", now)
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

// NextUpdate calculates the time for the next update and the associated
// wait time depending on the reset flag. If reset, the wait time starts
// at 5 minutes (300 sec), otherwise it is doubled before calculating the
// next update time.
func (mdl *Model) NextUpdate(ID int64, reset bool) error {
	// check for valid repository
	if mdl.inst == nil {
		return ErrModelNotAvailable
	}
	// set next wait time; wait time is randomized
	f := mdl.cfg.BalanceWait[1]
	r := rand.NormFloat64()*(0.25*f) + f
	if r < 1.0 {
		r = 1.0
	}
	wt := fmt.Sprintf("least(%f*waitCheck,%d)", r, int(mdl.cfg.BalanceWait[2]))
	if reset {
		wt = fmt.Sprintf("%d", int(mdl.cfg.BalanceWait[0]))
	}
	now := time.Now().Unix()
	_, err := mdl.inst.Exec(
		"update addr set lastCheck=?,waitCheck="+wt+
			",nextCheck=nextCheck+"+wt+" where id=?", now, ID)
	return err
}

// CloseAddress closes an address; no further usage (except spending)
func (mdl *Model) CloseAddress(ID int64) error {
	// check for valid repository
	if mdl.inst == nil {
		return ErrModelNotAvailable
	}
	// close address in model
	_, err := mdl.inst.Exec("update addr set stat=1, validTo=now() where id=?", ID)
	return err
}

// LockAddress locks an address after spending
func (mdl *Model) LockAddress(ID int64) error {
	// check for valid repository
	if mdl.inst == nil {
		return ErrModelNotAvailable
	}
	// lock address in model
	_, err := mdl.inst.Exec("update addr set stat=2 where id=?", ID)
	return err
}

// SyncAddress tags an address for immediate balance update
func (mdl *Model) SyncAddress(ID int64) error {
	// check for valid repository
	if mdl.inst == nil {
		return ErrModelNotAvailable
	}
	// enforce update now
	now := time.Now().Unix()
	_, err := mdl.inst.Exec("update addr set nextCheck=? where id=?", now, ID)
	return err
}

// GetAddressInfo returns basic info about an address
func (mdl *Model) GetAddressInfo(ID int64) (addr, coin string, balance, rate float64, err error) {
	// check for valid repository
	if mdl.inst == nil {
		return "", "", 0, 0, ErrModelNotAvailable
	}
	// get information about coin address
	row := mdl.inst.QueryRow("select coin,val,balance,rate from v_addr where id=?", ID)
	err = row.Scan(&coin, &addr, &balance, &rate)
	return
}

// GetAddressID returns the repository ID of an address
func (mdl *Model) GetAddressID(addr string) (id int64, err error) {
	// check for valid repository
	if mdl.inst == nil {
		return 0, ErrModelNotAvailable
	}
	// query ID
	row := mdl.inst.QueryRow("select id from addr where val=?", addr)
	err = row.Scan(&id)
	return
}

// AddrInfo holds information about an address
type AddrInfo struct {
	ID         int64   `json:"id"`         // id of address entry
	Status     int     `json:"status"`     // address status
	CoinName   string  `json:"coin"`       // name of coin
	CoinSymb   string  `json:"coinID"`     // coin symbol
	Account    string  `json:"account"`    // name of account
	AccntLabel string  `json:"accntLabel"` // account label
	Val        string  `json:"value"`      // address value
	Balance    float64 `json:"balance"`    // address balance
	Rate       float64 `json:"rate"`       // coin value (price per coin)
	RefCount   int     `json:"refCount"`   // number of transactions
	LastCheck  string  `json:"lastCheck"`  // last balance check
	NextCheck  string  `json:"nextCheck"`  // next balance check
	WaitCheck  int     `json:"waitCheck"`  // wait time between checks (seconds)
	LastTx     string  `json:"lastTx"`     // last used in a transaction
	ValidSince string  `json:"validSince"` // start of active period
	ValidUntil string  `json:"validUntil"` // end of active period
	Explorer   string  `json:"explorer"`   // URL to address in blockchain explorer
}

// GetAddress returns a list of active adresses
func (mdl *Model) GetAddresses(id, accnt, coin int64, all bool) (ai []*AddrInfo, err error) {
	// check for valid repository
	if mdl.inst == nil {
		return nil, ErrModelNotAvailable
	}
	// assemble WHERE clause
	clause := ""
	if !all {
		clause = " stat < 2"
	}
	addClause := func(id int64, field string) {
		if id != 0 {
			if len(clause) > 0 {
				clause += " and"
			}
			clause += fmt.Sprintf(" %s=%d", field, id)
		}
	}
	if id != 0 {
		addClause(id, "id")
	} else {
		if coin != 0 {
			addClause(coin, "coinId")
		}
		if accnt != 0 {
			addClause(accnt, "accntId")
		}
	}
	// assemble SELECT statement
	query := "select id,coin,coinName,val,balance,rate,stat,account,accountName," +
		"cnt,lastCheck,nextCheck,waitCheck,lastTx,validFrom,validTo from v_addr"
	if len(clause) > 0 {
		query += " where" + clause
	}
	query += " order by balance*rate desc,cnt desc"

	// get information about active addresses
	var rows *sql.Rows
	if rows, err = mdl.inst.Query(query); err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		addr := new(AddrInfo)
		var (
			last, next, tx sql.NullInt64
			from, to       sql.NullString
		)
		if err = rows.Scan(
			&addr.ID, &addr.CoinSymb, &addr.CoinName, &addr.Val, &addr.Balance,
			&addr.Rate, &addr.Status, &addr.AccntLabel, &addr.Account, &addr.RefCount,
			&last, &next, &addr.WaitCheck, &tx, &from, &to); err != nil {
			return
		}
		if last.Valid {
			addr.LastCheck = ""
			if last.Int64 > 0 {
				addr.LastCheck = time.Unix(last.Int64, 0).Format("02 Jan 06 15:04")
			}
		}
		if next.Valid {
			addr.NextCheck = ""
			if next.Int64 > 0 {
				addr.NextCheck = time.Unix(next.Int64, 0).Format("02 Jan 06 15:04")
			}
		}
		if tx.Valid {
			addr.LastTx = ""
			if tx.Int64 > 0 {
				addr.LastTx = time.Unix(tx.Int64, 0).Format("02 Jan 06 15:04")
			}
		}
		if from.Valid {
			addr.ValidSince = from.String
		}
		if to.Valid {
			addr.ValidUntil = to.String
		}
		// set explorer link
		if hdlr, ok := HdlrList[addr.CoinSymb]; ok {
			addr.Explorer = fmt.Sprintf(hdlr.explorer, addr.Val)
		}
		// add address info to list
		ai = append(ai, addr)
	}
	return
}

// UpdateBalance sets the new balance for an address
func (mdl *Model) UpdateBalance(ID int64, balance float64) error {
	// check for valid repository
	if mdl.inst == nil {
		return ErrModelNotAvailable
	}
	// update balance in model
	_, err := mdl.inst.Exec("update addr set balance=? where id=?", balance, ID)
	return err
}

// Incoming is an incoming transaction
type Incoming struct {
	Date    string
	Account string
	Coin    string
	Amount  float64
	Value   float64
}

// Incoming records funds received by an address
func (mdl *Model) Incoming(ID int64, amount float64) error {
	// check for valid repository
	if mdl.inst == nil {
		return ErrModelNotAvailable
	}
	// insert funding statement
	now := time.Now().Unix()
	_, err := mdl.inst.Exec("insert into incoming(firstSeen,addr,amount) values(?,?,?)", now, ID, amount)
	return err
}

// ListIncoming returns a list of recent incoming funds.
func (mdl *Model) ListIncoming(n int) (list []*Incoming, err error) {
	var rows *sql.Rows
	if rows, err = mdl.inst.Query(
		"select firstSeen,account,coin,amount,val from v_incoming order by firstSeen desc limit ?", n); err != nil {
		return
	}
	for rows.Next() {
		i := new(Incoming)
		var dt int64
		if err = rows.Scan(&dt, &i.Account, &i.Coin, &i.Amount, &i.Value); err != nil {
			return
		}
		i.Date = time.Unix(dt, 0).Format("2006-01-02 15:04:05")
		list = append(list, i)
	}
	return
}

// Fund represents an entry in the 'incoming' table (incoming fund)
type Fund struct {
	Seen   int64
	Addr   int64
	Amount float64
}

// GetFunds return a list of funds for given address
func (mdl *Model) GetFunds(addr int64) (list []*Fund, err error) {
	// check for valid repository
	if mdl.inst == nil {
		err = ErrModelNotAvailable
		return
	}
	var rows *sql.Rows
	if rows, err = mdl.inst.Query("select firstSeen,amount from incoming where addr=?", addr); err != nil {
		return
	}
	for rows.Next() {
		f := &Fund{Addr: addr}
		if err := rows.Scan(&f.Seen, &f.Amount); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return
}

//----------------------------------------------------------------------
// Assignement-related methods.
//----------------------------------------------------------------------

// CountAssignments returns the number of assignments between coins and
// accounts. An ID of "0" means "all".
func (mdl *Model) CountAssignments(coin, accnt int64) int {
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
	row := mdl.inst.QueryRow(query)
	var count int
	if err := row.Scan(&count); err != nil {
		logger.Printf(logger.ERROR, "CountAssign: "+err.Error())
		count = -1
	}
	return count
}

// ChangeAssignment adds or removes coin/account assignments
func (mdl *Model) ChangeAssignment(coin, accnt int64, add bool) (err error) {
	if add {
		_, err = mdl.inst.Exec("insert ignore into accept(coin,accnt) values(?,?)", coin, accnt)
	} else {
		_, err = mdl.inst.Exec("delete from accept where coin=? and accnt=?", coin, accnt)
	}
	return
}

//----------------------------------------------------------------------
// Account-related methods
//----------------------------------------------------------------------

// AccntInfo holds information about an account in the model.
type AccntInfo struct {
	ID    int64   `json:"id"`    // Id of account record
	Label string  `json:"label"` // account label
	Name  string  `json:"name"`  // account name
	Total float64 `json:"total"` // total balance of account (in fiat currency)
	NumTx int64   `json:"numTx"` // number of transactions for account
	Coins []*Item `json:"coins"` // (assigned) coins
}

// GetAccounts list all accounts with their total balance (in fiat currency)
func (mdl *Model) GetAccounts(id int64) (accnts []*AccntInfo, err error) {
	// check for valid repository
	if mdl.inst == nil {
		return nil, ErrModelNotAvailable
	}
	// assemble query
	query := `
		select
			account.id as id,
			account.label as label,
			account.name as name,
			sum(addr.balance*coin.rate) as total,
			sum(addr.refCnt) as refs
		from account
		left join addr on addr.accnt=account.id and addr.stat < 2
		left join coin on addr.coin=coin.id
		group by account.id`

	// select account information
	var rows *sql.Rows
	if rows, err = mdl.inst.Query(query); err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		// parse basic information
		ai := new(AccntInfo)
		var (
			total sql.NullFloat64
			refs  sql.NullInt64
		)
		if err = rows.Scan(&ai.ID, &ai.Label, &ai.Name, &total, &refs); err != nil {
			return
		}
		// filter for ID
		if id != 0 && ai.ID != id {
			continue
		}
		ai.Total = 0
		if total.Valid {
			ai.Total = total.Float64
		}
		ai.NumTx = 0
		if refs.Valid {
			ai.NumTx = refs.Int64
		}
		// get associated coins for account
		if ai.Coins, err = mdl.getItems(`
			select
  				coin.id as id,
  				coin.label as name,
  				(coin.id in (select coin from accept where accnt=?)) as status,
  				coin.rate as rate,
  				sum(addr.balance) as balance,
				count(addr.id) as addrs,
				coin.symbol as symbol,
				coin.logo as logo
			from coin
			left join addr on addr.coin = coin.id and addr.stat < 2 and addr.accnt = ?
			group by coin.id`, ai.ID, ai.ID); err != nil {
			return
		}
		// sort coins by descending fiat balance
		sort.Slice(ai.Coins, func(i, j int) bool {
			xi := ai.Coins[i].Dict["balance"]
			bi := -1.
			if xi != nil {
				bi = xi.(float64)
			}
			xj := ai.Coins[j].Dict["balance"]
			bj := -1.
			if xj != nil {
				bj = xj.(float64)
			}
			ri := ai.Coins[i].Dict["rate"].(float64)
			rj := ai.Coins[j].Dict["rate"].(float64)
			return rj*bj < ri*bi
		})
		// add to list
		accnts = append(accnts, ai)
	}
	// sort coins by descending fiat balance
	sort.Slice(accnts, func(i, j int) bool {
		return accnts[j].Total < accnts[i].Total
	})
	return
}

// GetAccountID returns repository ID of an account record.
func (mdl *Model) GetAccountID(label string) (accnt int64, err error) {
	// check for valid repository
	if mdl.inst == nil {
		return 0, ErrModelNotAvailable
	}
	// query ID
	row := mdl.inst.QueryRow("select id from accnt where label=?", label)
	err = row.Scan(&accnt)
	return
}

// NewAccount creates a new account with given label and name.
func (mdl *Model) NewAccount(label, name string) error {
	// check for valid repository
	if mdl.inst == nil {
		return ErrModelNotAvailable
	}
	// insert new record into model
	_, err := mdl.inst.Exec("insert into account(label,name) values(?,?)", label, name)
	return err
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
func (mdl *Model) NewTransaction(coin, account string) (tx *Transaction, err error) {
	// check for valid repository
	if mdl.inst == nil {
		return nil, ErrModelNotAvailable
	}
	// start repository transaction
	ctx := context.Background()
	var mdltx *sql.Tx
	if mdltx, err = mdl.inst.BeginTx(ctx, nil); err != nil {
		return
	}
	// get an address
	var addr string
	if addr, err = mdl.getUnusedAddress(mdltx, coin, account); err != nil {
		mdltx.Rollback()
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
		ValidTo:   now + int64(mdl.cfg.TxTTL),
	}
	var addrID int64
	var accnt sql.NullString
	row := mdltx.QueryRow("select id,coin,account from v_addr where val=?", addr)
	if err = row.Scan(&addrID, &tx.Coin, &accnt); err != nil {
		mdltx.Rollback()
		return
	}
	if accnt.Valid {
		tx.Accnt = accnt.String
	}
	// insert transaction into model
	if _, err = mdltx.Exec(
		"insert into tx(txid,addr,validFrom,validTo) values(?,?,?,?)",
		tx.ID, addrID, tx.ValidFrom, tx.ValidTo); err != nil {
		mdltx.Rollback()
		return
	}
	// increment ref counter in address
	if _, err = mdltx.Exec("update addr set refCnt=refCnt+1,lastTx=? where id=?", now, addrID); err != nil {
		mdltx.Rollback()
		return
	}
	// commit repository transaction
	err = mdltx.Commit()
	return
}

// GetTransactions returns a list of Tx instances for a given address
func (mdl *Model) GetTransactions(addrId, accntId, coinId int64) (txs []*Transaction, err error) {
	// check for valid repository
	if mdl.inst == nil {
		return nil, ErrModelNotAvailable
	}
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
	addClause(addrId, "addrId")
	addClause(accntId, "accntId")
	addClause(coinId, "coinId")

	// assemble SELECT statement
	query := "select txid,addr,coin,account,stat,validFrom,validTo from v_tx"
	if len(clause) > 0 {
		query += " where" + clause
	}
	query += " order by validFrom desc"

	// query model for transactions of given address
	var rows *sql.Rows
	if rows, err = mdl.inst.Query(query); err != nil {
		return
	}
	defer rows.Close()

	// assemble list
	for rows.Next() {
		tx := new(Transaction)
		if err = rows.Scan(&tx.ID, &tx.Addr, &tx.Coin, &tx.Accnt, &tx.Status, &tx.ValidFrom, &tx.ValidTo); err != nil {
			return
		}
		txs = append(txs, tx)
	}
	return
}

// GetTransaction returns the Tx instance for a given identifier
func (mdl *Model) GetTransaction(txid string) (tx *Transaction, err error) {
	// check for valid repository
	if mdl.inst == nil {
		return nil, ErrModelNotAvailable
	}
	// get information about transaction from model
	tx = new(Transaction)
	tx.ID = txid
	row := mdl.inst.QueryRow(
		"select addr,coin,account,stat,validFrom,validTo from v_tx where txid=?", txid)
	err = row.Scan(&tx.Addr, &tx.Coin, &tx.Accnt, &tx.Status, &tx.ValidFrom, &tx.ValidTo)
	return
}

// GetExpiredTransactions collects transactions that have expired.
// Returns a mapping between transaction and associated address.
func (mdl *Model) GetExpiredTransactions() (map[int64]int64, error) {
	// check for valid repository
	if mdl.inst == nil {
		return nil, ErrModelNotAvailable
	}
	// collect expired transactions
	t := time.Now().Unix()
	rows, err := mdl.inst.Query("select id,addr from tx where stat=0 and validTo<?", t)
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
func (mdl *Model) CloseTransaction(txID int64) error {
	// check for valid repository
	if mdl.inst == nil {
		return ErrModelNotAvailable
	}
	// close transaction in model
	_, err := mdl.inst.Exec("update tx set stat=1 where id=?", txID)
	return err
}

//----------------------------------------------------------------------
// Market-related methods
//----------------------------------------------------------------------

// UpdateRate sets the new exchange rate (in market base currency) for
// the given coin.
func (mdl *Model) UpdateRate(dt, coin, fiat string, rate float64) error {
	// check for valid repository
	if mdl.inst == nil {
		return ErrModelNotAvailable
	}
	// update rate in coin record
	if _, err := mdl.inst.Exec("update coin set rate=? where symbol=?", rate, coin); err != nil {
		return err
	}
	// update rate in rates table
	return mdl.SetRate(dt, coin, fiat, rate)
}

// GetRate returns a historical exchange rate for coin from rates table.
func (mdl *Model) GetRate(dt, coin, fiat string) (rate float64, err error) {
	row := mdl.inst.QueryRow("select rate from rates where dt=? and coin=? and fiat=?", dt, coin, fiat)
	if err = row.Scan(&rate); err != nil {
		rate = -1
	}
	return
}

// SetRate sets a historical exchange rate for coin in rates table.
func (mdl *Model) SetRate(dt, coin, fiat string, rate float64) error {
	// update rate in rates table
	_, err := mdl.inst.Exec(
		"insert into rates(dt,coin,rate,fiat) values(?,?,?,?)"+
			" on duplicate key update rate=(n*rate+?)/(n+1), n=n+1",
		dt, coin, rate, fiat, rate)
	return err
}
