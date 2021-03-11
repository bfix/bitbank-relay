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
	"bytes"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math/big"
	"strings"

	"github.com/bfix/gospel/bitcoin"
	"github.com/bfix/gospel/bitcoin/wallet"
)

func init() {
	handlers["bch"] = &BchHandler{}
}

type BchHandler struct {
}

// GetAddress returns a wallet address.
func (hdlr *BchHandler) GetAddress(ed *wallet.ExtendedData) (string, error) {

	pk, err := bitcoin.PublicKeyFromBytes(ed.Keydata)
	if err != nil {
		return "", err
	}
	switch ed.Version {
	case wallet.XpubVersion:
		return makeAddress(pk.Bytes(), 0), nil
	case wallet.YpubVersion:
		redeem := append([]byte(nil), 0)
		redeem = append(redeem, 0x14)
		kh := bitcoin.Hash160(pk.Bytes())
		redeem = append(redeem, kh...)
		return makeAddress(redeem, 5), nil
	}
	return "", fmt.Errorf("Unknown key data")
}

// makeAddress computes an address from public key for the Bitcoin-Cash network
func makeAddress(data []byte, version byte) string {

	// assemble payload
	payload := append([]byte(nil), version)
	kh := bitcoin.Hash160(data)
	payload = append(payload, kh...)

	b32 := base32.NewEncoding("qpzry9x8gf2tvdw0s3jn54khce6mua7l")
	addr := strings.Trim("bitcoincash:"+b32.EncodeToString(payload), "=")

	values := make([]byte, 54)
	copy(values, []byte{2, 9, 20, 3, 15, 9, 14, 3, 1, 19, 8, 0})
	copy(values[12:], bit5(payload))
	crc := polymod(values)
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, crc)
	return addr + strings.Trim(b32.EncodeToString(buf.Bytes()[3:]), "=")
}

// bit5 splits a byte array into 5-bit chunks
func bit5(data []byte) []byte {
	size := len(data) * 8
	v := new(big.Int).SetBytes(data)
	pad := size % 5
	if pad != 0 {
		v = new(big.Int).Lsh(v, uint(5-pad))
	}
	num := (size + 4) / 5
	res := make([]byte, num)
	for i := num - 1; i >= 0; i-- {
		res[i] = byte(v.Int64() & 31)
		v = new(big.Int).Rsh(v, 5)
	}
	return res
}

// polymod computes a CRC for 5-bit sequences
func polymod(values []byte) uint64 {
	var c uint64 = 1
	for _, d := range values {
		c0 := c >> 35
		c = ((c & 0x07ffffffff) << 5) ^ uint64(d)
		if c0&0x01 != 0 {
			c ^= 0x98f2bc8e61
		}
		if c0&0x02 != 0 {
			c ^= 0x79b76d99e2
		}
		if c0&0x04 != 0 {
			c ^= 0xf33e5fb3c4
		}
		if c0&0x08 != 0 {
			c ^= 0xae2eabe2a8
		}
		if c0&0x10 != 0 {
			c ^= 0x1e4f43e470
		}
	}
	return c ^ 1
}
