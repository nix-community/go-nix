-- name: InsertAttribute :exec
insert or replace into Attributes(parent, name, type, value) values (?, ?, ?, ?);

-- name: InsertAttributeWithContext :exec
insert or replace into Attributes(parent, name, type, value, context) values (?, ?, ?, ?, ?);

-- todo sqlc doesn't like the rowid column being included below
-- name: QueryAttribute :one
select type, value, context from Attributes where parent = ? and name = ?;

-- name: QueryAttributes :many
select name from Attributes where parent = ?;