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
	// Users
	CreateUser(context.Context, CreateUserRequest) (*User, error)
	GetUser(context.Context, string) (*User, error)
	ListUsers(context.Context) ([]User, error)
	// Invoices
	CreateInvoice(context.Context, CreateInvoiceRequest) (*Invoice, error)
	GetInvoice(context.Context, string) (*Invoice, error)
	ListInvoices(context.Context) ([]Invoice, error)
	// Media
	CreateMedia(context.Context, CreateMediaRequest) (*Media, error)
	GetMedia(context.Context, string) (*Media, error)
	GetMediaByInvoice(context.Context, string) (*Media, error)
	ListMedia(context.Context) ([]Media, error)

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

func (r *repo) Close() error {
	return r.db.Close()
}

func (r *repo) createSchema() error {
	const schema = `
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    pubkey TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS invoices (
    id TEXT PRIMARY KEY,
    payment_hash TEXT NOT NULL,
    sats INTEGER NOT NULL,
    paid BOOLEAN NOT NULL,
    provider TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    created_by TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS media (
    id TEXT PRIMARY KEY,
    invoice_id TEXT NOT NULL,
    size INTEGER NOT NULL,
    sum TEXT NOT NULL,
    original_filename TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    created_by TEXT NOT NULL,
    FOREIGN KEY (invoice_id) REFERENCES invoices(id)
);
	CREATE INDEX IF NOT EXISTS idx_user_pubkey ON users(pubkey);
	`

	if _, err := r.db.Exec(schema); err != nil {
		return err
	}

	return nil
}
