
-- create database
drop database if exists BB_Addresser;
create database BB_Addresser character set utf8mb4 collate utf8mb4_unicode_ci;
use BB_Addresser;

-- create user
drop user if exists 'bb_addr';
create user 'bb_addr'@'%' identified by 'bb_addr';
grant all on BB_Addresser.* to bb_addr;
flush privileges;

-- create tables
create table coin (
    id      integer auto_increment primary key,
    ref     integer not null,
    symbol  varchar(7),
    descr   varchar(63) default '',
    active  boolean default 0
);

create table addr (
    id      integer auto_increment primary key,
    coin    integer references coin(id) on delete cascade,
    val     varchar(127) not null,
    idx     integer not null,
    stat    integer default 0,
    firstSeen timestamp default current_timestamp,
    lastSeen timestamp null default null
);

