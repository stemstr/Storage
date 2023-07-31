package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	sub "github.com/stemstr/storage/internal/subscription"
)

func New(dbConnStr string) (*Repo, error) {
	db, err := sqlx.Connect("postgres", dbConnStr)
	if err != nil {
		return nil, fmt.Errorf("sqlx.Connect: %w", err)
	}

	// sqlx default is 0 (unlimited), while postgresql by default accepts up to 100 connections
	db.SetMaxOpenConns(80)

	// TODO: migrations
	_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS subscription (
	id SERIAL PRIMARY KEY,
	pubkey TEXT NOT NULL,
	days INTEGER NOT NULL,
	sats INTEGER NOT NULL,
	invoice_id TEXT NOT NULL,
	provider TEXT NOT NULL,
	status TEXT NOT NULL,
	lightning_invoice TEXT NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
	expires_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS pubkeyidx ON subscription(pubkey);
CREATE INDEX IF NOT EXISTS provideridx ON subscription(provider);
    `)
	if err != nil {
		return nil, fmt.Errorf("db.Exec schema: %w", err)
	}

	return &Repo{
		db: db,
	}, nil
}

type Repo struct {
	db *sqlx.DB
}

func (r *Repo) CreateSubscription(ctx context.Context, s sub.Subscription) (*sub.Subscription, error) {
	query, args, err := sqlx.Named(`INSERT INTO subscription (pubkey, days, sats, invoice_id, provider, status, lightning_invoice, expires_at) 
VALUES (:pubkey, :days, :sats, :invoice_id, :provider, :status, :lightning_invoice, :expires_at) RETURNING id;`, s)
	if err != nil {
		return nil, fmt.Errorf("sqlx.Named createSub: %w", err)
	}
	query = r.db.Rebind(query)
	var id int
	if err := r.db.Get(&id, query, args...); err != nil {
		return nil, fmt.Errorf("db.Get createSub: %w", err)
	}

	return r.getSubscription(ctx, int64(id))
}

func (r *Repo) GetActiveSubscriptions(ctx context.Context, pubkey string) ([]sub.Subscription, error) {
	const query = "SELECT * FROM subscription WHERE pubkey=$1 ORDER BY created_at DESC;"

	var subs []sub.Subscription
	if err := r.db.Select(&subs, query, pubkey); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("db.Get active sub: %w", err)
	}

	now := time.Now()
	var notExpired []sub.Subscription
	for _, s := range subs {
		if s.ExpiresAt.After(now) {
			notExpired = append(notExpired, s)
		}
	}

	return notExpired, nil
}

func (r *Repo) UpdateStatus(ctx context.Context, id int64, status sub.SubscriptionStatus) error {
	const query = `UPDATE subscription SET status=$2, updated_at=NOW() WHERE id=$1`
	params := []any{id, status}

	_, err := r.db.ExecContext(ctx, query, params...)
	if err != nil {
		return fmt.Errorf("db.Exec update sub: %w", err)
	}

	return nil
}

func (r *Repo) getSubscription(ctx context.Context, id int64) (*sub.Subscription, error) {
	const sql = "SELECT * FROM subscription WHERE id=$1;"

	var s sub.Subscription
	if err := r.db.Get(&s, sql, id); err != nil {
		return nil, fmt.Errorf("db.Get sub: %w", err)
	}

	return &s, nil
}
