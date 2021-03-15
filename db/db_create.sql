-- ---------------------------------------------------------------------
-- This file is part of 'bitbank-relay'.
-- Copyright (C) 2021 Bernd Fix >Y<
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
create user 'bb_relay'@'%' identified by 'bb_relay';
grant all on BB_Relay.* to bb_relay;
flush privileges;

-- ---------------------------------------------------------------------
-- create tables
-- ---------------------------------------------------------------------

create table coin (
    id      integer auto_increment primary key, -- database record id
    ref     integer not null,                   -- coin identifier
    symbol  varchar(7) not null,                -- coin symbol (lowercase)
    descr   varchar(63) default null,           -- coin name
    logo    text default null,                  -- coin logo (base64-encoded SVG)
    active  boolean default 0                   -- coin accepted?
);

create table account (
    id      integer auto_increment primary key, -- database record id
    ref     varchar(7) not null,                -- account short reference
    label   varchar(127) not null,              -- account label
    active  boolean default 0                   -- account accepted?
);

create table accept (
    accnt integer references account(id) on delete cascade,
    coin  integer references coin(id) on delete cascade
);

create view coins4account as select
    c.symbol as symb,
    c.descr as label,
    c.logo as logo,
    a.ref as ref,
    a.label as account
from
    coin c, account a, accept x 
where
    x.accnt = a.id and x.coin = c.id;

create table addr (
    id          integer auto_increment primary key,
    coin        integer references coin(id) on delete cascade,
    idx         integer not null,
    val         varchar(127) not null,
    stat        integer default 0,
    firstSeen   timestamp default current_timestamp,
    lastSeen    timestamp null default null
);

create table tx (
    id          integer auto_increment primary key,
    txid        varchar(64),
    addr        integer references addr(id) on delete cascade,
    accnt       integer references account(id) on delete cascade,
    validFrom   integer not null,
    validTo     integer not null
);
