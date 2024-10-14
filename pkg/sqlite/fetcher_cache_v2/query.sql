-- name: UpsertCache :exec
insert or replace into Cache(domain, key, value, timestamp) values (?, ?, ?, ?);

-- name: QueryCache :many
select value, timestamp from Cache where domain = ? and key = ?;