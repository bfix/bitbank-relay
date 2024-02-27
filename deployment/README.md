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

# Deployment

The deployment for `bitbank-relay` includes:

1. `bitbank-relay-web` executable (if build with GNU make, or the file
`web/web` if build manually)
2. `bitbank-relay-db` executable (if build with GNU make, or the file
`db/db` if build manually)
3. `config.json` (the genrated/edited configuration file)
4. an initialized relay database (MySQL (recommended) or SQLite3 (testing only))
5. integration into a website for use

Steps 4 and 5 are described in detail below; as an amendment some words
about running and maintaining the bitbank-relay can be found in the
"Operation" section.

## (Step 4) Relay database

The `config.json` specifies which database engine to use and how to access it.

* For a MySQL database:

```json
"database": {
    "mode": "mysql",
    "connect": "bb_relay:bb_relay@tcp(127.0.0.1:3306)/BB_Relay"
},
```

* For a SQLite3 database:

```json
"database": {
    "mode": "sqlite3",
    "connect": "relay.db"
},
```

Either database is initialized with a SQL script found in the `db/` folder:

* `db_create.mysql.sql` for MySQL database engine (adjust to your local env)
* `db_create.sqlite3.sql` for SQLite3 database file (add to deployment)

### Fill database with custom data

You need to customize the database with information about the accepted coins
and the accounts you want to support. Each account has one or more
crypto-currencies assigned to it; these are the coins that will be accepted
for an account.

#### Define the cryptocurrencies

**TL;DR**: To define the standard set of cryptocurrencies for bitbank-relay,
run the `db_init.sql` script found in the `db/` folder and you are done.

If you want to manually add cryptocurrencies, execute the following SQL
command on your database:

```sql
-- create list of supported coins
insert into coin(symbol,label) values
    ('btc',  'Bitcoin'),
    ('ltc',  'Litecoin'),
    ('doge', 'Dogecoin'),
    ('dash', 'Dash'),
    ('nmc',  'Namecoin'),
    ('dgb',  'Digibyte'),
    ('vtc',  'Vertcoin'),
    ('eth',  'Ethereum'),
    ('etc',  'Ethereum Classic'),
    ('zec',  'ZCash'),
    ('bch',  'Bitcoin Cash'),
    ('btg',  'Bitcoin Gold');
```

To add the coin logos to the database, change into the `db/` folder and run:

```bash
./bitbank-relay-db -c config.json logo import -i images
```

Coin logos have to be SVG files (minimized to keep their size smaller than
10kB) and their name must match the coin symbol in the database - otherwise
the import will fail.

If you add new coins make sure you create logos for the coins too. You can
add individual logo files by running:

```bash
./bitbank-relay-db -c config.json logo import -f images/coin.svg
```

#### Define accounts and accepted cryptocurrencies

**TL;DR**: This step is optional as you can define accounts and coin
acceptance in the GUI later.

To manually add accounts and acceptances, run the following SQL commands:

```sql
-- create list of accounts that accept coins
insert into account(label,name) values
    ('01', 'Campaign for ...'), ('02', 'Project ...');

-- create a map of coins for each account
insert into accept(coin,accnt) values
    (1, 1), (11, 1), (3, 2), ...;
```

The database is now set-up for productive use.

## (Step 5) Integration into a website for use

This is the tricky part... Usually you have to integrate the new relay
functionality into an existing site, so your choices on how to do that
can be limited. The following example integration only tries to explain
the two separate functionalities that the bitbank-relay offers:

1. **show a list of cryptocurrencies (as icons) with checkout links**: This is
usually embedded into an existing campaign/project/account page.

2. **show a checkout page displaying the receiving address (plain and QR)**:
This is usually a new page (as it is a new functionality).

The following **example** will use PHP to generate the webpages dynamically.
This is required as the relay might assign new coin addresses to projects on
certain conditions (see section "Operation").

### (1) show list of cryptocurrencies

The example uses a variable `{{label}}` that must match the value of the
`label` field in the database record for the current account.

```php
<!-- bitbank-relay -->
<div>
    <h3>Donate in cryptocurrency</h3>
    <p>Choose your coin:</p>
    <p>
        <?php $body = file_get_contents('http://172.23.0.42:4235/list?a={{label}}'); ?>
        <?php if ($body === false) : ?>
            <!-- Add your failsafe code: bitbank-relay is not available -->
        <?php else : ?>
            <?php
                $data = json_decode($body, true);
                foreach($data as $item) {
                    $coin = $item['symb'];
                    $label = $item['label'];
                    $logo = $item['logo'];
                    <!-- the href below refers to a new checkout page -->
                    echo '<a href="/checkout.php?a={{label}}&c=' .
                        $coin .
                        '"><img title="' .
                        $label .
                        '" src="data:image/svg+xml;base64,' .
                        $logo .
                        '" height="32"/></a>&nbsp;';
                }
            ?>
        <?php endif; ?>
    </p>
</div>
```

### (2) show checkout page

As said before this usually is a new webpage dedicated to show the receiving
coin address for a checkout. A minimal working example using PHP could look
like this:

```php
<html>
    <body>
        <?php
            $account = $_GET["a"];
            $coin = $_GET["c"];
            $txid = $_SESSION[$coin . $account];
            if ($txid == "") {
                $body = file_get_contents('http://172.23.0.42:4235/receive?a=' . $account . '&c=' . $coin);
            } else {
                $body = file_get_contents('http://172.23.0.42:4235/status?t=' . $txid);
            }
            $data = json_decode($body, true);
            $info = $data["coin"];
            $tx = $data["tx"];
            $txid = $tx["id"];
            if ($tx["status"] == 0) {
                $_SESSION[$coin . $account] = $txid;
            } else {
                $_SESSION[$coin . $account] = "";
            }
        ?>
        <div class="row">
            <div class="col-qr">
                <img src="<?php echo $data["qr"]; ?>" width="100%"/>
            </div>
            <div class="col-addr">
                <p>Send your coins to the following address:</p>
                <div class="addr">
                    <img
                        src="data:image/svg+xml;base64,<?php echo $info["logo"]; ?>"
                        height="32px" title="<?php echo $info["label"]; ?>"
                    />&nbsp;
                    <?php echo $tx["addr"]; ?>
                </div>
            </div>
        </div>
    </body>
</html>
```

# Operation

## Technical details

The two executables in the deployment set (`bitbank-relay-web` and
`bitbank-relay-db`) are each providing their own services. They share the
database and can run on separate systems/container/jails if desired.

You need to write startup/shutdown scripts for the services and the integration
of these scripts into your operating system yourself; the following description
only covers how to start the services from the command-line directly.

### bitbank-relay-web

This service provides a JSON-API for serving cryptocurrency addresses for
accounts on a website. It is started by the following command:

```bash
bitbank-relay-web -c config.json &
```

### bitbank-relay-db

This service provides a browser-based GUI for auditing and managing `bb_relay`.
It is started with the following command:

```bash
bitbank-relay-db -c config.json gui -l 0.0.0.0:8080 &
```
The service will listen on port `8080` and will accept any source IP
(`0.0.0.0`) for connections.

Once the service is started, visit the management webpages with a decent
modern browser.

If you run the pages behind a reverse proxy (e.g. nginx) on a path, don't
forget to use the `-p <prefix>` option!

## Crptocurrency details

### Managing addresses

Addresses only need to be managed manually if funds from an address are
about to transfered out (e.g. for cashing in).

Coin addresses have some model-related logic that is governed by its state.
The address state can be either OPEN(0), CLOSED(1) or LOCKED(2).

An address is generated, if a request for a specific coin/account pair does
not have an open address associated with it. A new address is in state OPEN.

If incoming funds on an address reach a certain custom threshold, the address
is automatically CLOSED and will not be used in client sessions.

Prior to transferring the funds from an address the address should be manually
LOCKED in the management GUI. N.B.: For safety reasons it is recommended to
only transfer funds out of an address if the address is LOCKED!

Adress balances are updated for OPEN and CLOSED addresses. If an open address
is used in a client session, the wait time until the next balance check is
set a custom minimum wait time. If the new balance (received funds) is greater
than the old balance, the wait time is also (re-)set to the minimum wait time;
otherwise the wait time is multiplied with a custom factor and capped at the
defined maximum wait time.
