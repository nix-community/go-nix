-- name: InsertCache :one
insert into BinaryCaches(url, timestamp, storeDir, wantMassQuery, priority)
values (?1, ?2, ?3, ?4, ?5)
on conflict (url)
do update set timestamp = ?2, storeDir = ?3, wantMassQuery = ?4, priority = ?5
returning id;

-- name: QueryCache :many
select id, storeDir, wantMassQuery, priority from BinaryCaches where url = ? and timestamp > ?;

-- name: InsertNar :exec
insert or replace into NARs(
    cache, hashPart, namePart, url, compression, fileHash, fileSize, narHash, narSize, refs, deriver, sigs, ca,
    timestamp, present
) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1);

-- name: InsertMissingNAR :exec
insert or replace into NARs(cache, hashPart, timestamp, present) values (?, ?, ?, 0);

-- name: QueryNar :many
select present, namePart, url, compression, fileHash, fileSize, narHash, narSize, refs, deriver, sigs, ca from NARs
where cache = ? and hashPart = ? and ((present = 0 and timestamp > ?) or (present = 1 and timestamp > ?));

-- name: InsertRealisation :exec
insert or replace into Realisations(cache, outputId, content, timestamp)
values (?, ?, ?, ?);

-- name: InsertMissingRealisation :exec
insert or replace into Realisations(cache, outputId, timestamp)
values (?, ?, ?);

-- name: QueryRealisation :many
select content from Realisations
where cache = ? and outputId = ?  and
(
    (content is null and timestamp > ?) or
    (content is not null and timestamp > ?)
);

-- name: QueryLastPurge :one
select value from LastPurge;

-- name: UpdateLastPurge :exec
insert or replace into LastPurge(dummy, value) values ('', ?);

-- name: PurgeNars :exec
delete from NARs where ((present = 0 and timestamp < ?) or (present = 1 and timestamp < ?));

