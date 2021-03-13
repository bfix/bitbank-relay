package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bfix/gospel/bitcoin"
	"github.com/bfix/gospel/bitcoin/wallet"
)

type Handler struct {
	coin int // coin identifier
	mode int              // adress mode (P2PKH, P2SH, ...)
	netw int // network (Main, Test, Reg)
	tree *wallet.HDPublic // HDKD for public keys
	pathTpl string           // path template for indexing addresses
}

func NewHandler(coin *CoinConfig, network int) (*Handler, error) {

	// compute base account address
	pk, err := wallet.ParseExtendedPublicKey(coin.Pk)
	if err != nil {
		return nil, err
	}
	pk.Data.Version = coin.GetXDVersion()

	// compute path tenplate for indexed addreses
	path := coin.Path
	for strings.Count(path, "/") < 4 {
		path += "/0"
	}
	path += "/%d"
	
	return &Handler{
		coin: wallet.GetCoinID(coin.Name),
		mode: coin.GetMode(),
		netw: network,
		pathTpl: path,
		tree: wallet.NewHDPublic(pk, coin.Path),
	}, nil
}

func (hdlr *Handler) GetAddress(idx int) (string, error) {

	// get extended public key for indexed address
	epk, err := hdlr.tree.Public(fmt.Sprintf(hdlr.pathTpl, idx))
	if err != nil {
		return "", err
	}
	ed := epk.Data

	// get public key data
	pk, err := bitcoin.PublicKeyFromBytes(ed.Keydata)
	if err != nil {
		return "", err
	}
	return wallet.MakeAddress(pk, hdlr.coin, hdlr.mode, hdlr.netw), nil
}

//----------------------------------------------------------------------
// shared blockchain APIs
//----------------------------------------------------------------------

// Blockcyper works for: BTC, LTC, DASH, DOGE, ETH
// Checks if an address is used (#tx > 0)
func Blockcypher(coin, addr string) (bool, error) {
	query := fmt.Sprintf("https://api.blockcypher.com/v1/%s/main/addrs/%s", coin, addr)
	resp, err := http.Get(query)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	var data map[string]interface{}
	if err = json.Unmarshal(body, &data); err != nil {
		return false, err
	}
	val, ok := data["n_tx"]
	if !ok {
		return false, fmt.Errorf("No 'n_tx' attribute")
	}
	n, ok := val.(uint64)
	if !ok {
		return false, fmt.Errorf("Invalid 'n_tx' type")
	}
	return n > 0, nil
}
