-- ---------------------------------------------------------------------
-- This file is part of 'bitbank-relay'.
-- Copyright (C) 2021 Bernd Fix   >Y<
--
-- 'bitbank-relay' is free software: you can redistribute it and/or modify
-- it under the terms of the GNU Affero General Public License as published
-- by the Free Software Foundation, either version 3 of the License,
-- or (at your option) any later version.
--
-- 'bitbank-relay' is distributed in the hope that it will be useful,
-- but WITHOUT ANY WARRANTY; without even the implied warranty of
-- MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
-- Affero General Public License for more details.
--
-- You should have received a copy of the GNU Affero General Public License
-- along with this program.  If not, see <http://www.gnu.org/licenses/>.
--
-- SPDX-License-Identifier: AGPL3.0-or-later
-- ---------------------------------------------------------------------

-- ---------------------------------------------------------------------
-- create tables
-- ---------------------------------------------------------------------

-- coin describes a cryptocurrency accepted by the relay
create table coin (
    id     integer     primary key,     -- database record id
    symbol varchar(7)  not null unique, -- coin symbol (lowercase short name)
    label  varchar(63) default null,    -- coin long name / description
    logo   text        default null,    -- coin logo (base64-encoded SVG)
    rate   float       default 0.0      -- market data for coin
);

-- account is a receiver for cryptocoins
create table account (
    id    integer      primary key,     -- database record id
    label varchar(7)   not null unique, -- account label
    name  varchar(127) default null     -- account name
);

-- accept list all account/coin pairs that can be processed
create table accept (
    accnt integer references account(id) on delete cascade, -- reference to account
    coin  integer references coin(id) on delete cascade,    -- reference to coin
    unique (accnt, coin)                                    -- unique combinations
);

-- addr is a cryptocurrency address that can receive coins
create table addr (
    id        integer      primary key,                              -- database record id
    coin      integer      references coin(id) on delete cascade,    -- associated coin
    idx       integer,                                               -- BIP32/39/44 address index
    val       varchar(127) not null,                                 -- address as string
    stat      integer      default 0,                                -- status:
                                                                     --  0 = open (valid; ready to be used)
                                                                     --  1 = closed (address was used; don't use again)
                                                                     --  2 = removed (after balance is transfered)
    accnt     integer      references account(id) on delete cascade, -- reference to account
    refCnt    integer      default 0,                                -- reference count (transactions)
    balance   float        default 0.0,                              -- address balance
    lastCheck integer      default 0,                                -- last balance check timestamp
    dirty     boolean      default false,                            -- address used after check
    lastTx    integer      default 0,                                -- timestamp of last tx usage
    validFrom timestamp    default current_timestamp,                -- address life-span start
    validTo   timestamp    null default null                         -- address life-span end
);

-- transaction
create table tx (
    id        integer     primary key,                           -- database record id
    txid      varchar(32) unique,                                -- 256-bit transaction identifier
    addr      integer     references addr(id) on delete cascade, -- reference to address used in transaction
    stat      integer     default 0,                             -- status:
                                                                 --  0 = pending
                                                                 --  1 = expired
    validFrom integer     not null,                              -- transaction life-span (start)
    validTo   integer     not null                               -- transaction life-span (end)
);

-- ---------------------------------------------------------------------
-- create views
-- ---------------------------------------------------------------------

-- id-less view on account/coin pairs that are accepted
create view v_coin_accnt as select
    c.id     as coinId,   -- coin database ID
    c.symbol as coin,     -- coin symbol
    c.label  as label,    -- coin name/label
    c.logo   as logo,     -- coin logo (as b64-encoded SVG)
    c.rate   as rate,     -- current market price for coin
    a.id     as accntid,  -- account database ID
    a.label  as account
from
    coin c, account a, accept x
where
    x.accnt = a.id and x.coin = c.id;

-- view on address records
create view v_addr as select
    a.id        as id,           -- address database ID
    c.id        as coinId,       -- coin database ID
    c.symbol    as coin,         -- coin ticker symbol
    c.label     as coinName,     -- coin name
    a.val       as val,          -- address string
    a.balance   as balance,      -- balance in coins
    c.rate      as rate,         -- current market price for coin
    a.stat      as stat,         -- address status
    b.id        as accntId,      -- account database ID
    b.label     as account,      -- account label/slug
    b.name      as accountName,  -- account name
    a.refCnt    as cnt,          -- ref. count for address
    a.lastCheck as lastCheck,    -- timestamp of last balance check
    a.validFrom as validFrom,    -- address life-span (start)
    a.validTo   as validTo       -- address life-span (end)
from
    addr a
inner join
    coin c on c.id = a.coin
left join
    account b on b.id = a.accnt;

-- id-less view on a transaction
create view v_tx as select
    t.txid      as txid,      -- transaction ID
    a.id        as addrId,    -- addrress database ID
    a.val       as addr,      -- address string
    c.id        as coinId,    -- coin database ID
    c.label     as coin,      -- coin name
    b.id        as accntId,   -- account database ID
    b.name      as account,   -- account name
    t.stat      as stat,      -- transaction status
    t.validFrom as validFrom, -- transaction life-span (start)
    t.validTo   as validTo    -- transaction life-span (end)
from
    tx t, addr a, account b, coin c
where
    t.addr = a.id and a.accnt = b.id and a.coin = c.id;
