# Bitbank - Relay

(c) 2021-2022 Bernd Fix <brf@hoi-polloi.org>   >Y<

bitbank-relay is free software: you can redistribute it and/or modify it
under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License,
or (at your option) any later version.

bitbank-relay is distributed in the hope that it will be useful, but
WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.

SPDX-License-Identifier: AGPL3.0-or-later

# Configuration

The configuration program `bitbank-relay-configurator` will use a template
configuration `config-template.json` (either embedded or external) and will
store the result in a file named `config.json` for productive use:

```bash
bitbank-relay-configurator [-n <network>] [-i <template>] [-o <output>]
```

The command-line options are:

* **-n &lt;network&gt;**: [`main`,`test`] The blockchain network to use; the
default is 'main' (N.B.: 'test' will not work on all coins!)

* **-i &lt;template&gt;**: Name of the external configuration template. If
not used, the embedded template will be used.

* **-o &lt;output&gt;**: Name of the rsulting configuration file. Defaults
to `config.json`.

You can export the embedded configuration template to the current directory by
using the special option `-export`:

```bash
bitbank-relay-configurator -export
```

You might want to modify the template and use it with the `-i` option during
a configuration run...

## Template

Make yourself familiar with the template as you might want to change settings
(either in the template or in the generated `config.json`) to customize the
software for your needs.

There are five top-level sections named `service`, `database`, `balancer`,
`market` and `coins`

### "service"

```json
"service": {
    "listen": "localhost:80",
    "epoch": 300,
    "logFile": "relay.log",
    "logLevel": "DBG",
    "logRotate": 288
}
```

* **listen** specifies the listen address for the JSON API. You can specify a
port and what external addresses to listen to.

* **epoch** is the time between two "heart beats" in seconds; periodic tasks
define their frequency in epochs.

* **logFile** specifies the name of a log file; if it is missing, logging will
be sent to the console.

* **logLevel** defines the minimium level of a message to get logged (`DBG`,
`INFO`, `WARN`, `ERROR`).

* **logRotate** defines the number of epochs after which a logfile is rotated.

### "model"

```json
"model": {
    "mode": "mysql",
    "connect": "bb_relay:bb_relay@tcp(127.0.0.1:3306)/BB_Relay"
}
```

* **mode** defines which database engine to use (`mysql` or `sqlite3`).

* **connect** specifies the connect string for the database; its format and
content depends on the specific database engine used.

### "balancer"

```json
"balancer": {
    "accountLimit": 10000,
    "rescan": 48,
    "apikeys": {
        "blockchair": ""
    }
}
```

* **accountLimit** specifies the amount of fiat currency (see `market`) when
an address is closed automatically (not shown again as a receiving address).

* **rescan** defines the number of epochs between address balance checks.

* **apikeys** is reserved for defining API keys for services that return
address balances for specific cryptocurrencies. Currently only an API key
for "BlockChair.com" is used (and you only need it if you have more than
1440 balance requests a day; watch the log files for messages that indicate
you need an API key for that service).

### "market"

```json
"market": {
    "fiat": "EUR",
    "rescan": 72,
    "apikey": ""
}
```

* **fiat** is the standard name for the fiat currency you want to use
internally and should be specified in capital letters. This is the currency
used in the `balancer` section for `accountLimit` field.

* **rescan** is the number of epochs between market price retreival.

* **apikey** is the API token you received after registering with
[CoinAPI.io](https://coinapi.io).

### "coins"

```json
"coins": [
    {
        "symb": "btc",
        "path": "m/49'/0'/0'",
        "mode": "P2SH",
        "pk": "",
        "addr": ""
    },
    :
]
```

The coins supported are listed in an array; the template file contains all
supported coins. If you don't want to use certain coins, just delete them
from the list. Each coin has the following fields:

* **symb** is the short name of the coin (in lower-case letters).

* **path** is the HD base path to the account; you usually don't have to
change that value during customization.

* **mode** defines the address format by specifying the transaction mode
(currently either `P2PKH` or `P2SH`).

* **pk** is the `xpub` key of the base account

* **addr** is the first address withn an account (index 0). This value is used
to verify a coin setup at start-up.

## Semi-automatic configuration

This assumes that you are going to setup a new multi-coin HD wallet / Trezor
device. This is the standard procedure for most setups.

### Generate new HD wallet data

As a security measure it is recommended to run this step on an air-gapped
computer or at least in a safe environment (e.g. in a secure system like
[Tails](https://tails.boum.org)). Just copy the `configurator` executable
to the target system; the configurator program will use the embedded
`config-template.json` file and generates a new `config.json` file you need
to deploy later on. Run the configurator:

```bash
./configurator | tee config.log
```

First you will be asked for a passphrase; make sure you use a long and safe
input. Do not reuse existing passphrases, but create a new one especially
for this purpose.

The program will output information to the console; the above command captures
the output to a file `config.log` that can be printed for safe-keeping. Anyone
with access to this information will be able to make transactions! Make sure
you delete the file after printing.

### Initialize HD wallet / Trezor device

Part of the printed information (`config.log`) are the 24 seed words used to
setup a HD wallet or Trezor device.

After setting up the wallet you should check the addresses listed in
`config.log` for all cryptocurrencies to match the addresses generated by the
wallet. Please check at lease two addresses per coin to make sure the setup
worked correctly.

## Manual configuration

If you have an existing HD wallet / Trezor device you want to use without a
full reset with new keys, you can manually setup the `config.json` file - but
it is a time-consuming task.

Copy the `config-template.json` to a new `config.json` and open the new file
in an editor. You need to fill in data for the `pk` and `addr` fields in all
entries in the `coins` section:

```json
"coins": [
    {
        "symb": "btc",
        "path": "m/49'/0'/0'",
        "mode": "P2SH",
        "pk": "",
        "addr": ""
    },

```

Use the wallet UI to extract the `xpub` key for an account and insert the
result in the `pk` field. Also put the first address of an account into
the `addr` field.

### Using a Trezor device

You can use the `trezorctl` utility to generate the required information
from an initialized Trezor device.

**Example**: Generate the info for Bitcoin (`btc`) by running the
following commands:

```bash
trezorctl get-public-node -t p2shsegwit -n "m/49'/0'/0'"
```

The value for `pk` is returned in the line starting with "xpub:"

```bash
trezorctl get-address -t p2shsegwit -n "m/49'/0'/0'/0/0"
```

The command prints the value for `addr`.
