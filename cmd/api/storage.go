package main

import (
	"context"
	"io"
)

type storageProvider interface {
	Save(ctx context.Context, data io.Reader, sum string) error
	Get(ctx context.Context, sum string) (io.Reader, error)
}
