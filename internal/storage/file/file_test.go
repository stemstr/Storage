package file

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stemstr/storage/internal/storage"
)

func TestSave(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "files*")
	assert.NoError(t, err)
	defer os.Remove(tempDir)

	store, err := New(map[string]string{
		"media_dir": tempDir,
	})
	assert.NoError(t, err)

	var (
		ctx               = context.Background()
		filename          = "test.txt"
		data              = []byte("testdata")
		sum               = fmt.Sprintf("%x", sha256.Sum256(data))
		src               = bytes.NewReader(data)
		expectedMediaPath = filepath.Join(sum, filename)
	)

	mediaPath, err := store.Save(ctx, src, storage.Options{
		Filename: filename,
		Sha256:   sum,
	})
	assert.NoError(t, err)
	assert.Equal(t, expectedMediaPath, mediaPath)
}

func TestGet(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "files*")
	assert.NoError(t, err)
	defer os.Remove(tempDir)

	store, err := New(map[string]string{
		"media_dir": tempDir,
	})
	assert.NoError(t, err)

	var (
		ctx      = context.Background()
		filename = "test.txt"
		data     = []byte("testdata")
		sum      = fmt.Sprintf("%x", sha256.Sum256(data))
		src      = bytes.NewReader(data)
	)

	// First write a file
	_, err = store.Save(ctx, src, storage.Options{
		Filename: filename,
		Sha256:   sum,
	})
	assert.NoError(t, err)

	// Ensure we can read it from the expected location
	{
		f, err := store.Get(ctx, sum, filename)
		assert.NoError(t, err)

		b, err := ioutil.ReadAll(f)
		assert.NoError(t, err)
		assert.Equal(t, data, b)
	}

	// Ensure we can read it using a random filename
	// (A file is uniquely identified by its hash)
	{
		f, err := store.Get(ctx, sum, "alternate.txt")
		assert.NoError(t, err)

		b, err := ioutil.ReadAll(f)
		assert.NoError(t, err)
		assert.Equal(t, data, b)
	}
}
