package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"relay/lib"

	"github.com/bfix/gospel/logger"
)

//======================================================================
// handle GUI requests
//======================================================================

// RootData holds all information to render the root view.
type RootData struct {
	Accounts []*lib.AccntInfo `json:"accounts"` // list of active accounts
}

// handle main entry page
func rootHandler(w http.ResponseWriter, r *http.Request) {
	// collect information for the dashboard
	dd := new(RootData)

	// collect account info
	var err error
	if dd.Accounts, err = mdl.GetAccounts(0); err != nil {
		io.WriteString(w, "ERROR: "+err.Error())
		return
	}
	// show dashboard
	renderPage(w, dd, "root")
}

//----------------------------------------------------------------------

// AccountData holds the information needed to render an "account" page.
type AccountData struct {
	Accnt *lib.AccntInfo  `json:"accnt"` // info about account
	Coins []*lib.CoinInfo `json:"coins"` // list of assigned coins
}

func accountHandler(w http.ResponseWriter, r *http.Request) {
	// show coins assigned to account
	query := r.URL.Query()
	ad := new(AccountData)

	label := query["l"][0]
	id, err := mdl.GetAccountID(label)
	if err != nil {
		logger.Printf(logger.ERROR, "error getting account id: %s", err)
		return
	}
	list, err := mdl.GetAccounts(id)
	if err != nil {
		logger.Printf(logger.ERROR, "error getting account list: %s", err)
		return
	}
	ad.Accnt = list[0]

	req := "http://" + cfg.Service.Listen + fmt.Sprintf("/list/?a=%s", label)
	if verbose {
		logger.Printf(logger.DBG, ">>> GET %s", req)
	}
	resp, err := http.Get(req)
	if err != nil {
		logger.Printf(logger.ERROR, "error making http request: %s", err)
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Printf(logger.ERROR, "error reading http response: %s", err)
		return
	}
	if verbose {
		logger.Printf(logger.DBG, "<< %s", string(body))
	}
	if err = json.Unmarshal(body, &ad.Coins); err != nil {
		logger.Printf(logger.ERROR, "error unmarshalling http response: %s", err)
		logger.Println(logger.ERROR, string(body))
		return
	}

	// show account page
	renderPage(w, ad, "account")
}

//----------------------------------------------------------------------

type TxResponse struct {
	Error string           `json:"error,omitempty"`
	Tx    *lib.Transaction `json:"tx"`
	Qr    string           `json:"qr"`
	Coin  *lib.CoinInfo    `json:"coin"`
}

// PayData holds the information needed to render an "payment" page.
type PayData struct {
	Accnt *lib.AccntInfo `json:"accnt"` // info about account
	Tx    *TxResponse    `json:"resp"`  // service response
}

func payHandler(w http.ResponseWriter, r *http.Request) {
	// show coins assigned to account
	query := r.URL.Query()
	pd := new(PayData)

	accnt := query["a"][0]
	id, err := mdl.GetAccountID(accnt)
	if err != nil {
		logger.Printf(logger.ERROR, "error getting account id: %s", err)
		return
	}
	list, err := mdl.GetAccounts(id)
	if err != nil {
		logger.Printf(logger.ERROR, "error getting account list: %s", err)
		return
	}
	pd.Accnt = list[0]

	req := "http://" + cfg.Service.Listen + fmt.Sprintf("/receive/?a=%s&c=%s", accnt, query["c"][0])
	if verbose {
		logger.Printf(logger.DBG, ">>> GET %s", req)
	}
	resp, err := http.Get(req)
	if err != nil {
		logger.Printf(logger.ERROR, "error making http request: %s", err)
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Printf(logger.ERROR, "error reading http response: %s", err)
		return
	}
	if verbose {
		logger.Printf(logger.DBG, "<< %s", string(body))
	}
	if err = json.Unmarshal(body, &pd.Tx); err != nil {
		logger.Printf(logger.ERROR, "error unmarshalling http response: %s", err)
		logger.Println(logger.ERROR, string(body))
		return
	}

	// show account page
	renderPage(w, pd, "checkout")
}

//======================================================================
// Helper methods
//======================================================================

// render a webpage with given data and template reference
func renderPage(w io.Writer, data interface{}, page string) {
	// create content section
	t := tpl.Lookup(page)
	if t == nil {
		io.WriteString(w, "No template '"+page+"' found")
		return
	}
	content := new(bytes.Buffer)
	if err := t.Execute(content, data); err != nil {
		io.WriteString(w, err.Error())
		return
	}
	// emit final page
	t = tpl.Lookup("main")
	if t == nil {
		io.WriteString(w, "No main template found")
		return
	}
	if err := t.Execute(w, content.String()); err != nil {
		io.WriteString(w, err.Error())
	}
}
