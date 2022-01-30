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

# Running the configuration program

The configuration program `bitbank-relay-configurator` will use a template
configuration `config-template.json` (either embedded or external) and will
store the result in a file named `config.json` for productive use:

```bash
bitbank-relay-configurator [-m <mode>] [-n <network>] [-i <template>] [-o <output>]
```

All command-line options are optional:

* **-m &lt;mode&gt;**: [`trezor`,`seed`] Configuration mode:
    * `trezor`: Automatic from Trezor device (should be used if a Trezor One
      or Trezor Model T is available). This is the default mode.
    * `seed`: Semi-automatic from passphrase for use with multi-coin software
      wallet(s)

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

# Configuration template `config-template.json`

Make yourself familiar with the template as you might want to change settings
(in a copy of the template or a generated configuration named `config.json`)
to customize the software for your needs.

There are four top-level sections named `service`, `model`, `handler` and
`coins`.

## "service"

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

## "model"

```json
"model": {
    "dbEngine": "mysql",
    "dbConnect": "bb_relay:bb_relay@tcp(127.0.0.1:3306)/BB_Relay",
    "balanceWait": [
        300,
        2,
        604800
    ],
    "txTTL": 900
},
```

* **dnEngine** defines which database engine to use (`mysql` or `sqlite3`).

* **dbConnect** specifies the connect string for the database; its format and
content depends on the specific database engine used.

* **balanceWait** defines the delay between balance checks and contains three
values: the first specifies the minimum wait time (for new or updated addresses;
defaults to 5 minutes). The third value is the maximum wait time (a week). The
second number specifies the mean value for the factor to increase wait time in
case of unchanged addresses; it is randomized within bounds to spread balance
checks across time.

* **txTTL** is the time-to-live for transactions (defaults to 15 minutes)

## "handler"

```json
"handler": {
    "blockchain": {
        "blockchair.com": {
            "apiKey": "",
            "rateLimits": [ 5, 30, 0, 1440 ]
        },
        "cryptoid.info": {
            "apiKey": "",
            "coolTime": 10.0
        },
        "btgexplorer.com": {
            "apiKey": "",
            "rates": [ 5, 30, 0, 1440 ]
        },
        "zcha.in": {
            "apiKey": "",
            "rates": [ 5, 30, 0, 1440 ]
        },
        "blockscout.com": {
            "apiKey": "",
            "rates": [ 0, 6, 0, 1440 ]
        }
    },
    "market": {
        "fiat": "EUR",
        "rescan": 72,
        "service": {
            "coinapi.io": {
                "apiKey": ""
            }
        }
    }
},
```

The `handler`section has two subsections, describing the handlers used for
blockchain and market requests (balance checks, rate updates, etc.)
`bitbank-relay` comes with a number of services defined:

### "blockchain"

* **apiKey** specifies an API keys for the service. Some serivces offer free
plans but allow only a limited number of requests. If you need a better plan
(usually paid), you should ask for an API key to make use of that.

* **rates** defines the rate limits imposed by a service. It is an array of
integer values corresponding to the rate limits "per second", "per minute",
"per hour", "per day" and "per week". A value of "0" means the rate limit is
not defined and requests are limited by the next higher (non-null) rate limit.

* **coolTime** defines a fixed wait time between two requests; it is used as
an alternative to the `rates`definition.

### "market"

* **fiat** is the standard name for the fiat currency you want to use
internally and should be specified in capital letters. This is the currency
used in the `balancer` section for `accountLimit` field.

* **rescan** is the number of epochs between market price retreival.

* **service**

Defines a list of market services; the parameters of a service (`apiKey`,
`rates` and `coolTime`) have the same meaning than the corresponding 
parameters in `blockchain` handlers.

To retrieve market data, you need a free registration and an API token
you receive after registering with [CoinAPI.io](https://coinapi.io).

## "coins"

```json
"coins": [
    {
        "symb": "btc",
        "path": "m/49'/0'/0'",
        "mode": "P2SH",
        "pk": "",
        "addr": "",
        "explorer": "<explorer URL pattern for address like https://.../%s>",
        "accountLimit": 10000,
        "blockchain": "<handler name>"
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

* **explorer** defines the URL pattern for viewing an address with a blockchain
explorer.

* **accountLimit** defines how much funds (in fiat currency) an address can hold
(accumulate), before it is automatically closed.

* **blockchain** specifies the name of the blockchain handler that is used to
manage/query address balances for the coin.

# Automatic configuration

This assumes that you are going to setup an existing and initialized Trezor
device. The wallet should only be used for `bitbank-relay`, so the easiest way
is to use a passphrase-protected "hidden wallet" on the Trezor. This is
the recommended way and standard procedure for most setups. The Trezor device
can then easily be used to manage all incoming funds in a single wallet like
Trezor Suite that runs locally on your machine.

Make sure a single Trezor device (Trezor One or Trezor Model T) is connected
via USB to the computer. When run, the Trezor One will require you to enter a
pin to unlock the device.

## Pin/password entry

If you have protected the Trezor with a pin and/or a password, some functions
might require you to authorize access by providing pin and/or password.

The configurator will work on the command line; if a pin is required, the
Trezor device will display the pin matrix and the console shows a 3x3 matrix
with numbers too:

```
+---+---+---+
| 7 | 8 | 9 |
+---+---+---+
| 4 | 5 | 6 |
+---+---+---+
| 1 | 2 | 3 |
+---+---+---+

PIN? █
```

Locate the first pin digit position on the Trezor and enter the corresponding
number shown on the console display. Proceed until all pin digits are entered.
Press ENTER to submit the entry.

The layout of the numbers to enter corresponds with the ordering of the number
keys on the number block of your keyboard (on the right side). If you have
enabled `NUM_LOCK` on your keyboard, you can easily enter the pin using the
positions on the number block.

# Semi-automatic configuration

This assumes that you are going to setup (a) new multi-coin HD wallet(s) to
manage incoming funds. The data needed (either seed or xpubs) is generated by
the configuration program with the `seed` mode (command line option `-m seed`).

## Generate new HD wallet data

As a security measure it is recommended to run this step on an air-gapped
computer or at least in a safe environment (e.g. in a secure system like
[Tails](https://tails.boum.org)). Just copy the `configurator` executable
to the target system; the configurator program will use the embedded
`config-template.json` file and generates a new `config.json` file you need
to deploy later on. Run the configurator:

```bash
./configurator -m seed | tee config.log
```

First you will be asked for a passphrase; make sure you use a long and safe
input. Do not reuse existing passphrases, but create a new one especially
for this purpose.

The program will output information to the console; the above command captures
the output to a file `config.log` that can be printed for safe-keeping. Anyone
with access to this information will be able to make transactions! Make sure
you delete the file after printing.

## Initialize HD wallet

Part of the printed information (`config.log`) are the 24 seed words used to
setup a HD wallet.

After setting up the wallet you should check the addresses listed in
`config.log` for all cryptocurrencies to match the addresses generated by the
wallet. Please check at lease two addresses per coin to make sure the setup
worked correctly.

# Manual configuration

If you have an existing HD wallet you want to use without a full reset with
new keys, you can manually setup the `config.json` file - but it is a
time-consuming task.

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
        "addr": "",
        :
    },
    :
```

Use the wallet UI to extract the `xpub` key for an account and insert the
result in the `pk` field. Also put the first address of an account into
the `addr` field.
