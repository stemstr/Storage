package filesystem

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrite(t *testing.T) {
	tempDir := t.TempDir()
	defer os.Remove(tempDir)

	var (
		ctx  = context.Background()
		fs   = New()
		data = []byte("testdata")
		sum  = fmt.Sprintf("%x", sha256.Sum256(data))
		path = filepath.Join(tempDir, sum+".txt")
	)

	err := fs.Write(ctx, path, data)
	assert.NoError(t, err)
}

func TestWriteDirNotExists(t *testing.T) {
	tempDir := "./foo/bar/baz"

	var (
		ctx  = context.Background()
		fs   = New()
		data = []byte("testdata")
		sum  = fmt.Sprintf("%x", sha256.Sum256(data))
		path = filepath.Join(tempDir, sum+".txt")
	)

	err := fs.Write(ctx, path, data)
	assert.NoError(t, err)

	assert.NoError(t, os.RemoveAll(tempDir))
}

func TestGet(t *testing.T) {
	tempDir := t.TempDir()
	defer os.Remove(tempDir)

	var (
		ctx  = context.Background()
		fs   = New()
		data = []byte("testdata")
		sum  = fmt.Sprintf("%x", sha256.Sum256(data))
		path = filepath.Join(tempDir, sum+".txt")
	)

	// First write a file
	err := fs.Write(ctx, path, data)
	assert.NoError(t, err)

	b, err := fs.Read(ctx, path)
	assert.NoError(t, err)
	assert.Equal(t, data, b)
}
