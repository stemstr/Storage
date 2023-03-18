package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Invoice struct {
	ID          string
	PaymentHash string
	Sats        int64
	Paid        bool
	Provider    string
	CreatedAt   time.Time
	CreatedBy   string
}

type CreateInvoiceRequest struct {
	PaymentHash string
	Paid        bool
	Sats        int64
	Provider    string
	CreatedBy   string
}

func (r *repo) CreateInvoice(ctx context.Context, req CreateInvoiceRequest) (*Invoice, error) {
	stmt, err := r.db.PrepareContext(ctx, "INSERT INTO invoices (id, payment_hash, sats, paid, provider, created_at, created_by) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	invoice := &Invoice{
		ID:          uuid.New().String(),
		PaymentHash: req.PaymentHash,
		Sats:        req.Sats,
		Paid:        req.Paid,
		Provider:    req.Provider,
		CreatedAt:   time.Now(),
		CreatedBy:   req.CreatedBy,
	}

	if _, err := stmt.ExecContext(ctx, invoice.ID, invoice.PaymentHash, invoice.Sats, invoice.Paid, invoice.Provider, invoice.CreatedAt, invoice.CreatedBy); err != nil {
		return nil, err
	}

	return invoice, nil
}

func (r *repo) GetInvoice(ctx context.Context, id string) (*Invoice, error) {
	var invoice Invoice

	row := r.db.QueryRowContext(ctx, "SELECT id, payment_hash, sats, paid, provider, created_at, created_by FROM invoices WHERE id=?", id)

	err := row.Scan(&invoice.ID, &invoice.PaymentHash, &invoice.Sats, &invoice.Paid, &invoice.Provider, &invoice.CreatedAt, &invoice.CreatedBy)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &invoice, nil
}

func (r *repo) ListInvoices(ctx context.Context) ([]Invoice, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, payment_hash, sats, paid, provider, created_at, created_by FROM invoices")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []Invoice

	for rows.Next() {
		var invoice Invoice
		if err := rows.Scan(&invoice.ID, &invoice.PaymentHash, &invoice.Sats, &invoice.Paid, &invoice.Provider, &invoice.CreatedAt, &invoice.CreatedBy); err != nil {
			return nil, err
		}
		invoices = append(invoices, invoice)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return invoices, nil
}
