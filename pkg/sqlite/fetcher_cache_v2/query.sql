-- name: UpsertCache :exec
insert or replace into Cache(domain, key, value, timestamp) values (?, ?, ?, ?);

-- name: QueryCache :one
select value, timestamp from Cache where domain = ? and key = ?;