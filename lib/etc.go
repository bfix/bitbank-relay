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
	"encoding/hex"

	"golang.org/x/crypto/sha3"
)

func init() {
	handlers["etc"] = &EtcHandler{}
}

type EtcHandler struct {
	BaseHandler
}

// GetAddress returns a wallet address.
func (hdlr *EtcHandler) GetAddress(idx int) (string, error) {
	pk, _, err := hdlr.getPublicKey(idx)
	if err != nil {
		return "", err
	}
	pkData := pk.Q.Bytes(false)
	hsh := sha3.NewLegacyKeccak256()
	hsh.Write(pkData[1:])
	val := hsh.Sum(nil)
	return "0x" + hex.EncodeToString(val[12:]), nil
}
