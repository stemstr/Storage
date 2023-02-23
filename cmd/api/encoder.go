package main

import (
	"context"
)

type encoderProvider interface {
	EncodeMP3(ctx context.Context, mediaPath, mediaType, name string) (string, error)
}
