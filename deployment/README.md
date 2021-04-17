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

# Deployment

The deployment for `bb_relay` includes:

1. `web` executable (from the `web/` folder)
2. `config.json` (the genrated/edited configuration file)
3. an initialized relay database (MySQL or SQLite3)
4. integration into a website for use

Steps 3 and 4 are described in detail below; as an amendment some words
about running and maintaining the bitbank-relay can be found in the
"Operation" section.

## (3) Relay database

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
* `fb_create.sqlite3.sql` for SQLite3 database file (add to deployment)

### Fill database with custom data

You need to customize the database with information about the accepted coins
and the accounts you want to support. Each account has one or more
crypto-currencies assigned to it; these are the coins that will be accepted
for an account.

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

-- create list of accounts that accept coins
insert into account(label,name) values
    ('01', 'Campaign for ...'), ('02', 'Project ...');

-- create a map of coins for each account
insert into accept(coin,accnt) values
    (1, 1), (11, 1), (3, 2), ...;
```

To add the coin logos to the database, change into the `db/` folder and run:

```bash
./db -c config.json logo import -i images
```

Coin logos have to be SVG files (minimized to keep their size smaller than
10kB) and their name must match the coin symbol in the database - otherwise
the import will fail.

If you add new coins make sure you create logos for the coins too. You can
add individual logo files by running:

```bash
./db -c config.json logo import -f images/coin.svg
```

The database is now set-up for productive use.

## (4) Integration into a website for use

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

(to be described)
