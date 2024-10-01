-- name: RegisterValidPath :exec
insert into ValidPaths (path, hash, registrationTime, deriver, narSize, ultimate, sigs, ca)
values (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdatePathInfo :exec
update ValidPaths set narSize = ?, hash = ?, ultimate = ?, sigs = ?, ca = ? where path = ?;

-- name: AddReference :exec
insert or replace into Refs (referrer, reference) values (?, ?);

-- name: QueryPathInfo :one
select id, hash, registrationTime, deriver, narSize, ultimate, sigs, ca from ValidPaths where path = ?;

-- name: QueryReferences :many
select path from Refs join ValidPaths on reference = id where referrer = ?;

-- name: QueryReferrers :many
select path from Refs join ValidPaths on referrer = id where reference = (select vp.id from ValidPaths as vp where vp.path = ?);

-- name: InvalidatePath :exec
delete from ValidPaths where path = ?;

-- name: AddDerivationOutput :exec
insert or replace into DerivationOutputs (drv, id, path) values (?, ?, ?);

-- name: QueryValidDerivers :many
select v.id, v.path from DerivationOutputs d join ValidPaths v on d.drv = v.id where d.path = ?;

-- name: QueryDerivationOutputs :many
select id, path from DerivationOutputs where drv = ?;

-- name: QueryPathFromHashPart :one
select path from ValidPaths where path >= ? limit 1;