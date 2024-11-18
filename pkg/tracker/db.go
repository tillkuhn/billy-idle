package tracker

import (
	// provides access to Files embedded in the running Go program.
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"

	_ "modernc.org/sqlite" // import sqlite driver
	sqlite3 "modernc.org/sqlite/lib"
)

// Package embed provides access to Files embedded in the running Go program.
//
//go:embed init-db.sql
var initSQL string

// initDB initializes SQLite DB in local filesystem
func initDB(opts *Options) (*sqlx.DB, error) {
	if err := os.MkdirAll(opts.AppDir(), os.ModePerm); err != nil {
		return nil, fmt.Errorf("cannot create app dir %s: %w", opts.AppDir(), err)
	}
	dbFile := filepath.Join(opts.AppDir(), "db.sqlite3")

	log.Printf("🥫 Open database file=%s sqlite=%s", dbFile, sqlite3.SQLITE_VERSION)
	db, err := sqlx.Open("sqlite", dbFile)
	if err != nil {
		return nil, fmt.Errorf("cannot open db %s: %w", dbFile, err)
	}

	opts.ClientID, err = os.Hostname()
	if err != nil {
		return nil, err
	}

	// drop table if exists t; insert into t values(42), (314);
	var dropStmt string
	if opts.DropCreate {
		dropStmt = "DROP TABLE IF EXISTS track;\n"
	}
	if _, err = db.Exec(dropStmt + initSQL); err != nil {
		return nil, err
	}

	return db, nil
}
