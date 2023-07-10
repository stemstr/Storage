package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type Filesystem interface {
	Read(ctx context.Context, path string) ([]byte, error)
	Remove(ctx context.Context, path ...string)
	Write(ctx context.Context, path string, data []byte) error
}

func New() Filesystem {
	return &filesystem{}
}

type filesystem struct{}

func (fs *filesystem) Read(ctx context.Context, path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (fs *filesystem) Remove(ctx context.Context, path ...string) {
	for _, filePath := range path {
		os.Remove(filePath)
	}
}

func (fs *filesystem) Write(ctx context.Context, path string, data []byte) error {
	dir := filepath.Dir(path)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return fmt.Errorf("create dir: %w", err)
		}
	}

	return os.WriteFile(path, data, 0644)
}
