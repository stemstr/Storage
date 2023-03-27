package blob

import (
	"context"
	"fmt"
)

type S3 interface {
	Write(ctx context.Context, path string, data []byte) error
}

func New() S3 {
	return &s3Client{}
}

type s3Client struct{}

func (*s3Client) Write(ctx context.Context, path string, data []byte) error {
	fmt.Printf("TODO: s3.Write")
	return nil
}
