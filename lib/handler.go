package lib

import (
	"fmt"

	"github.com/bfix/gospel/bitcoin/wallet"
)

type Handler interface {
	GetAddress(*wallet.ExtendedData) (string, error)
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
