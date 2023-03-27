package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Media struct {
	ID        string
	InvoiceID string
	Size      int64
	Sum       string
	Mimetype  string
	Waveform  []int
	CreatedAt time.Time
	CreatedBy string
}

type CreateMediaRequest struct {
	Size      int64
	Sum       string
	Mimetype  string
	Waveform  []int
	CreatedBy string
}

func (r *repo) CreateMedia(ctx context.Context, req CreateMediaRequest) (*Media, error) {
	stmt, err := r.db.PrepareContext(ctx, "INSERT INTO media (id, invoice_id, size, sum, mimetype, waveform, created_at, created_by) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	media := &Media{
		ID:        uuid.New().String(),
		InvoiceID: "FIXME",
		Size:      req.Size,
		Sum:       req.Sum,
		Mimetype:  req.Mimetype,
		Waveform:  req.Waveform,
		CreatedAt: time.Now(),
		CreatedBy: req.CreatedBy,
	}

	waveformData, err := json.Marshal(req.Waveform)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal: %w", err)
	}

	if _, err := stmt.ExecContext(ctx, media.ID, media.InvoiceID, media.Size, media.Sum, media.Mimetype, waveformData, media.CreatedAt, media.CreatedBy); err != nil {
		return nil, err
	}

	return media, nil
}

func (r *repo) GetMedia(ctx context.Context, sum string) (*Media, error) {
	var media Media

	row := r.db.QueryRowContext(ctx, "SELECT id, invoice_id, size, sum, mimetype, waveform, created_at, created_by FROM media WHERE sum=?", sum)

	var waveformData []byte
	err := row.Scan(&media.ID, &media.InvoiceID, &media.Size, &media.Sum, &media.Mimetype, &waveformData, &media.CreatedAt, &media.CreatedBy)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(waveformData, &media.Waveform); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}

	return &media, nil
}

func (r *repo) ListMedia(ctx context.Context) ([]Media, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, invoice_id, size, sum, mimetype, created_at, created_by FROM media")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mediaList []Media

	for rows.Next() {
		var media Media
		if err := rows.Scan(&media.ID, &media.InvoiceID, &media.Size, &media.Sum, &media.Mimetype, &media.CreatedAt, &media.CreatedBy); err != nil {
			return nil, err
		}
		mediaList = append(mediaList, media)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return mediaList, nil
}
