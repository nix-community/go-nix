CREATE TABLE BinaryCaches (
    id        integer primary key autoincrement not null,
    url       text unique not null,
    timestamp integer not null,
    storeDir  text not null,
    wantMassQuery integer not null,
    priority  integer not null
);
CREATE TABLE sqlite_sequence(name,seq);
CREATE TABLE NARs (
    cache            integer not null,
    hashPart         text not null,
    namePart         text,
    url              text,
    compression      text,
    fileHash         text,
    fileSize         integer,
    narHash          text,
    narSize          integer,
    refs             text,
    deriver          text,
    sigs             text,
    ca               text,
    timestamp        integer not null,
    present          integer not null,
    primary key (cache, hashPart),
    foreign key (cache) references BinaryCaches(id) on delete cascade
);
CREATE TABLE Realisations (
    cache integer not null,
    outputId text not null,
    content blob, -- Json serialisation of the realisation, or null if the realisation is absent
    timestamp        integer not null,
    primary key (cache, outputId),
    foreign key (cache) references BinaryCaches(id) on delete cascade
);
CREATE TABLE LastPurge (
    dummy            text primary key,
    value            integer
);
