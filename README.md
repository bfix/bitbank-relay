[![Go Report Card](https://goreportcard.com/badge/github.com/bfix/bitbank-relay)](https://goreportcard.com/report/github.com/bfix/bitbank-relay)
[![GoDoc](https://godoc.org/github.com/bfix/bitbank-relay?status.svg)](https://godoc.org/github.com/bfix/bitbank-relay)

# Bitbank - Relay (bb_relay)

(c) 2021 Bernd Fix <brf@hoi-polloi.org>   >Y<

bb_relay is free software: you can redistribute it and/or modify it
under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License,
or (at your option) any later version.

bb_relay is distributed in the hope that it will be useful, but
WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.

SPDX-License-Identifier: AGPL3.0-or-later

# Introduction

The `bb_relay` software enables individuals and small organizations to accept
cryptocurrencies (Bitcoin, Ethereum and ten other Altcoins) on their webpage.

To manage the received coins it is highly recommended to use a multi-coin
HD wallet with optional [Trezor support](https://trezor.io).

# Build

If you want to build the software yourself, you need `Go v1.16+` that can be
[downloaded here](https://golang.org/dl/). Make sure you setup Go-related
environment variables as described in the Go documentation.

After you have cloned the repository to your local machine (and every time
you pull a newer version), you should update the dependencies for `bb_relay`:

```bash
go mod tidy
```

To build the three components (configurator, db and web):

```bash
cd configurator
go build
cd ../db
go build
cd ../web
go build
cd ..
```

# Configuration

You need to configure/setup the `bb_relay` package in parallel with the
multi-coin HD wallet you want to use to manage incoming crypto funds. You can
either do this semi-automatically or manually.

The steps are described in a separate
[README](https://github.com/bfix/bitbank-relay/tree/master/configurator).

# Deployment

The deployment for `bb_relay` includes:

* `web` executable (from the `web/` folder)
* `config.json` (the genrated/edited configuration file)
* an initialized relay database (MySQL or SQLite3)
* integration into a website for use

A detailed description can be found in a separate
[README](https://github.com/bfix/bitbank-relay/tree/master/deployment).

# Testing

(to be described)
