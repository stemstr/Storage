package quotestore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type QuoteStore interface {
	Create(context.Context, *Request) (*Quote, error)
	Get(ctx context.Context, id string) (*Quote, error)
	Close() error
}

func New(dbFile string) (QuoteStore, error) {
	if dbFile == "" {
		return nil, fmt.Errorf("must set quote_db")
	}

	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}

	s := quoteStore{
		dbFile: dbFile,
		DB:     db,
	}

	if err := s.createSchema(); err != nil {
		return nil, err
	}

	return &s, nil
}

type quoteStore struct {
	dbFile string
	DB     *sql.DB
}

func (s *quoteStore) Create(ctx context.Context, req *Request) (*Quote, error) {
	const insert = `insert into quote(id, pk, c_hash, c_size) values(?, ?, ?, ?)`

	stmt, err := s.DB.Prepare(insert)
	if err != nil {
		return nil, err
	}

	id := uuid.New().String()

	_, err = stmt.Exec(
		id,
		req.Pubkey,
		req.ContentHash,
		req.ContentSize,
	)
	if err != nil {
		return nil, err
	}

	return s.Get(ctx, id)
}

func (s *quoteStore) Get(ctx context.Context, id string) (*Quote, error) {
	const query = `SELECT pk, c_hash, c_size, r_hash, paid, created_at FROM quote WHERE id=?;`

	var (
		quote = Quote{
			ID: id,
		}
		paid      int
		createdAt string
	)

	err := s.DB.QueryRow(query, id).Scan(
		&quote.Pubkey,
		&quote.ContentHash,
		&quote.ContentSize,
		&quote.PaymentHash,
		&paid,
		&createdAt,
	)
	switch err {
	case nil:
		// break
	case sql.ErrNoRows:
		return nil, fmt.Errorf("quote with id=%s not found", id)
	default:
		return nil, fmt.Errorf("failed to query quote: %w", err)
	}

	quote.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode created_at timestamp: %w", err)
	}

	return &quote, nil
}

func (s *quoteStore) Close() error {
	return s.DB.Close()
}

func (s *quoteStore) createSchema() error {
	const schema = `
	CREATE TABLE IF NOT EXISTS quote(
		id TEXT NOT NULL, 
		pk TEXT NOT NULL,
		c_hash TXT NOT NULL,
		c_size INTEGER NOT NULL,
		r_hash TEXT,
		paid INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_id ON quote(id);
	CREATE INDEX IF NOT EXISTS idx_pk ON quote(pk);`

	if _, err := s.DB.Exec(schema); err != nil {
		return err
	}

	return nil
}
