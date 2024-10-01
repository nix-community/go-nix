CREATE TABLE ValidPaths (
    id               integer primary key autoincrement not null,
    path             text unique not null,
    hash             text not null, -- base16 representation
    registrationTime integer not null,
    deriver          text,
    narSize          integer,
    ultimate         integer, -- null implies "false"
    sigs             text, -- space-separated
    ca               text -- if not null, an assertion that the path is content-addressed; see ValidPathInfo
);
CREATE TABLE sqlite_sequence(name,seq);
CREATE TABLE Refs (
    referrer  integer not null,
    reference integer not null,
    primary key (referrer, reference),
    foreign key (referrer) references ValidPaths(id) on delete cascade,
    foreign key (reference) references ValidPaths(id) on delete restrict
);
CREATE INDEX IndexReferrer on Refs(referrer);
CREATE INDEX IndexReference on Refs(reference);
CREATE TRIGGER DeleteSelfRefs before delete on ValidPaths
  begin
    delete from Refs where referrer = old.id and reference = old.id;
  end;
CREATE TABLE DerivationOutputs (
    drv  integer not null,
    id   text not null, -- symbolic output id, usually "out"
    path text not null,
    primary key (drv, id),
    foreign key (drv) references ValidPaths(id) on delete cascade
);
CREATE INDEX IndexDerivationOutputs on DerivationOutputs(path);
