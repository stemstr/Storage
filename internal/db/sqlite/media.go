package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Media struct {
	ID               string
	InvoiceID        string
	Size             int64
	Sum              string
	OriginalFilename string
	CreatedAt        time.Time
	CreatedBy        string
}

type CreateMediaRequest struct {
	InvoiceID        string
	Size             int64
	Sum              string
	OriginalFilename string
	CreatedBy        string
}

func (r *repo) CreateMedia(ctx context.Context, req CreateMediaRequest) (*Media, error) {
	stmt, err := r.db.PrepareContext(ctx, "INSERT INTO media (id, invoice_id, size, sum, original_filename, created_at, created_by) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	media := &Media{
		ID:               uuid.New().String(),
		InvoiceID:        req.InvoiceID,
		Size:             req.Size,
		Sum:              req.Sum,
		OriginalFilename: req.OriginalFilename,
		CreatedAt:        time.Now(),
		CreatedBy:        req.CreatedBy,
	}

	if _, err := stmt.ExecContext(ctx, media.ID, media.InvoiceID, media.Size, media.Sum, media.OriginalFilename, media.CreatedAt, media.CreatedBy); err != nil {
		return nil, err
	}

	return media, nil
}

func (r *repo) GetMedia(ctx context.Context, id string) (*Media, error) {
	var media Media

	row := r.db.QueryRowContext(ctx, "SELECT id, invoice_id, size, sum, original_filename, created_at, created_by FROM media WHERE id=?", id)

	err := row.Scan(&media.ID, &media.InvoiceID, &media.Size, &media.Sum, &media.OriginalFilename, &media.CreatedAt, &media.CreatedBy)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &media, nil
}

func (r *repo) GetMediaByInvoice(ctx context.Context, id string) (*Media, error) {
	var media Media

	row := r.db.QueryRowContext(ctx, "SELECT id, invoice_id, size, sum, original_filename, created_at, created_by FROM media WHERE invoice_id=?", id)

	err := row.Scan(&media.ID, &media.InvoiceID, &media.Size, &media.Sum, &media.OriginalFilename, &media.CreatedAt, &media.CreatedBy)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &media, nil
}

func (r *repo) ListMedia(ctx context.Context) ([]Media, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, invoice_id, size, sum, original_filename, created_at, created_by FROM media")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mediaList []Media

	for rows.Next() {
		var media Media
		if err := rows.Scan(&media.ID, &media.InvoiceID, &media.Size, &media.Sum, &media.OriginalFilename, &media.CreatedAt, &media.CreatedBy); err != nil {
			return nil, err
		}
		mediaList = append(mediaList, media)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return mediaList, nil
}
