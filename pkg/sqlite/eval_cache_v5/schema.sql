create table if not exists Attributes (
    parent      integer not null,
    name        text,
    type        integer not null,
    value       text,
    context     text,
    primary key (parent, name)
);