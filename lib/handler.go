package lib

import (
	"fmt"
	"strings"

	"github.com/bfix/gospel/bitcoin"
	"github.com/bfix/gospel/bitcoin/wallet"
)

type Handler interface {
	Init(string, *wallet.ExtendedPublicKey) error
	GetAddress(idx int) (string, error)
}

type BaseHandler struct {
	tree *wallet.HDPublic
	path string
}

func (hdlr *BaseHandler) Init(path string, epk *wallet.ExtendedPublicKey) error {
	hdlr.path = path
	hdlr.tree = wallet.NewHDPublic(epk, path)
	return nil
}

func (hdlr *BaseHandler) getPublicKey(idx int) (*bitcoin.PublicKey, int, error) {
	// adjust path length
	path := hdlr.path
	for strings.Count(path, "/") < 4 {
		path += "/0"
	}
	// get extended public key
	epk, err := hdlr.tree.Public(path + fmt.Sprintf("/%d", idx))
	if err != nil {
		return nil, 0, err
	}
	ed := epk.Data

	// get public key data
	pk, err := bitcoin.PublicKeyFromBytes(ed.Keydata)
	return pk, int(ed.Version), err
}

var (
	handlers = make(map[string]Handler)
)

func GetHandler(coin string) (Handler, error) {
	hdlr, ok := handlers[coin]
	if !ok {
		return nil, fmt.Errorf("No handler for coin '%s'", coin)
	}
	return hdlr, nil
}
