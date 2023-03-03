package main

import (
	"context"

	quotes "github.com/stemstr/storage/internal/quotestore"
)

type quoteDBProvider interface {
	Create(context.Context, *quotes.Request) (*quotes.Quote, error)
	Get(ctx context.Context, id string) (*quotes.Quote, error)
}
