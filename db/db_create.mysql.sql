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

-- create database
drop database if exists BB_Relay;
create database BB_Relay character set utf8mb4 collate utf8mb4_unicode_ci;
use BB_Relay;

-- create user
drop user if exists 'bb_relay';
create user 'bb_relay' @'%' identified by 'bb_relay';
grant all on BB_Relay.* to bb_relay;
flush privileges;

-- ---------------------------------------------------------------------
-- create tables
-- ---------------------------------------------------------------------

-- coin describes a cryptocurrency accepted by the relay
create table coin (
    -- database record id
    id integer auto_increment primary key,

    -- coin symbol (lowercase short name)
    symbol varchar(7) not null,

    -- coin long name / description
    label varchar(63) default null,

    -- coin logo (base64-encoded SVG)
    logo text default null,

    -- market data for coin
    rate float default 0.0
);

-- account is a receiver for cryptocoins
create table account (
    -- database record id
    id integer auto_increment primary key,

    -- account label
    label varchar(7) not null,

    -- account name
    name varchar(127) default null
);

-- accept list all account/coin pairs that can be processed
create table accept (
    accnt integer references account(id) on delete cascade,
    coin integer references coin(id) on delete cascade,
    unique key (accnt, coin) 
);

-- id-less view on account/coin pairs that are accepted
create view coins4account as select
    c.id as coinId,
    c.symbol as coin,
    c.label as label,
    c.logo as logo,
    c.rate as rate,
    a.id as accntid,
    a.label as account
from
    coin c, account a, accept x
where
    x.accnt = a.id and x.coin = c.id;

-- addr is a cryptocurrency address that can receive coins
create table addr (
    -- database record id
    id integer auto_increment primary key,

    -- associated coin
    coin integer references coin(id) on delete cascade,

    -- BIP32/39/44 address index
    idx integer,

    -- address as string
    val varchar(127) not null,

    -- status:
    --  0 = open (ready to be used)
    --  1 = closed (address was used; don't use again)
    --  2 = removed (after balance is transfered)
    stat integer default 0,

    -- reference to account
    accnt integer references account(id) on delete cascade,

    -- reference count (transactions)
    refCnt integer default 0,

    -- address balance
    balance float default 0.0,
    lastCheck integer default 0,

    -- address life-span
    validFrom timestamp default current_timestamp,
    validTo timestamp null default null
);

-- view on address records
create view v_addr as select
    a.id as id,
    c.id as coinId,
    c.symbol as coin,
    c.label as coinName,
    a.val as val,
    a.balance as balance,
    c.rate as rate,
    a.stat as stat,
    b.id as accntId,
    b.label as account,
    b.name as accountName,
    a.refCnt as cnt,
    a.lastCheck as lastCheck,
    a.validFrom as validFrom,
    a.validTo as validTo
from
    addr a
inner join
    coin c on c.id = a.coin
left join
    account b on b.id = a.accnt;

-- transaction
create table tx (
    -- database record id
    id integer auto_increment primary key,

    -- 256-bit transaction identifier
    txid varchar(64),

    -- reference to address used in transaction
    addr integer references addr(id) on delete cascade,

    -- status:
    --  0 = pending
    --  1 = expired
    stat integer default 0,

    -- transaction life-span
    validFrom integer not null,
    validTo integer not null
);

-- id-less view on a transaction
create view v_tx as select
    t.txid as txid,
    a.id as addrId,
    a.val as addr,
    c.id as coinId,
    c.label as coin,
    b.id as accntId,
    b.name as account,
    t.stat as stat,
    t.validFrom as validFrom,
    t.validTo as validTo
from
    tx t, addr a, account b, coin c
where
    t.addr = a.id and a.accnt = b.id and a.coin = c.id;
