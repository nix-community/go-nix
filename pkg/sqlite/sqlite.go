package sqlite

import (
	"database/sql"
	"fmt"

	// enable the sqlite3 driver.
	_ "github.com/mattn/go-sqlite3"
	"github.com/nix-community/go-nix/pkg/sqlite/binary_cache_v6"
	"github.com/nix-community/go-nix/pkg/sqlite/eval_cache_v5"
	"github.com/nix-community/go-nix/pkg/sqlite/fetcher_cache_v2"
	"github.com/nix-community/go-nix/pkg/sqlite/nix_v10"
)

func BinaryCacheV6(dsn string) (*sql.DB, *binary_cache_v6.Queries, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	return db, binary_cache_v6.New(db), nil
}

func EvalCacheV5(dsn string) (*sql.DB, *eval_cache_v5.Queries, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	return db, eval_cache_v5.New(db), nil
}

func FetcherCacheV2(dsn string) (*sql.DB, *fetcher_cache_v2.Queries, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	return db, fetcher_cache_v2.New(db), nil
}

func NixV10(dsn string) (*sql.DB, *nix_v10.Queries, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	return db, nix_v10.New(db), nil
}
