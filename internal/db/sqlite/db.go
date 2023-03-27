package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type DB interface {
	// Media
	CreateMedia(context.Context, CreateMediaRequest) (*Media, error)
	GetMedia(context.Context, string) (*Media, error)
	ListMedia(context.Context) ([]Media, error)

	DB() *sql.DB
	Close() error
}

func New(dbFile string) (DB, error) {
	if dbFile == "" {
		return nil, fmt.Errorf("must set db_file")
	}
	if _, err := os.Stat(dbFile); errors.Is(err, os.ErrNotExist) {
		fmt.Printf("creating db file %v\n", dbFile)
		f, err := os.Create(dbFile)
		if err != nil {
			return nil, err
		}
		f.Close()
	}

	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}

	r := repo{
		dbFile: dbFile,
		db:     db,
	}

	if err := r.createSchema(); err != nil {
		return nil, err
	}

	return &r, nil
}

type repo struct {
	dbFile string
	db     *sql.DB
}

func (r *repo) DB() *sql.DB {
	return r.db
}

func (r *repo) Close() error {
	return r.db.Close()
}

func (r *repo) createSchema() error {
	const schema = `
CREATE TABLE IF NOT EXISTS media (
    id TEXT PRIMARY KEY,
    invoice_id TEXT,
    size INTEGER NOT NULL,
    sum TEXT NOT NULL,
    mimetype TEXT NOT NULL,
    waveform TEXT,
    created_at DATETIME NOT NULL,
    created_by TEXT NOT NULL
);
	CREATE INDEX IF NOT EXISTS idx_media_sum ON media(sum);
	`

	if _, err := r.db.Exec(schema); err != nil {
		return err
	}

	return nil
}
