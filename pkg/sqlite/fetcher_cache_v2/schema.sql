create table if not exists Cache (
    domain    text not null,
    key       text not null,
    value     text not null,
    timestamp integer not null,
    primary key (domain, key)
);