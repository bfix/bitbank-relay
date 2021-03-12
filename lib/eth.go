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

package lib

import (
	"fmt"

	"github.com/bfix/gospel/bitcoin"
	"github.com/bfix/gospel/bitcoin/wallet"
)

func init() {
	handlers["eth"] = &EthHandler{}
}

type EthHandler struct {
}

// GetAddress returns a wallet address.
func (hdlr *EthHandler) GetAddress(ed *wallet.ExtendedData) (string, error) {

	pk, err := bitcoin.PublicKeyFromBytes(ed.Keydata)
	if err != nil {
		return "", err
	}
	switch ed.Version {
	case wallet.XpubVersion:
		return wallet.MakeAddress(pk, 60, wallet.AddrP2PKH, wallet.AddrMain), nil
	case wallet.YpubVersion:
		return wallet.MakeAddress(pk, 60, wallet.AddrP2SH, wallet.AddrMain), nil
	}
	return "", fmt.Errorf("Unknown key data")
}
