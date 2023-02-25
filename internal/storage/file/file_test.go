package file

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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
		ctx  = context.Background()
		data = []byte("testdata")
		sum  = fmt.Sprintf("%x", sha256.Sum256(data))
		src  = bytes.NewReader(data)
	)

	err = store.Save(ctx, src, sum)
	assert.NoError(t, err)
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
		ctx  = context.Background()
		data = []byte("testdata")
		sum  = fmt.Sprintf("%x", sha256.Sum256(data))
		src  = bytes.NewReader(data)
	)

	// First write a file
	err = store.Save(ctx, src, sum)
	assert.NoError(t, err)

	f, err := store.Get(ctx, sum)
	assert.NoError(t, err)

	b, err := ioutil.ReadAll(f)
	assert.NoError(t, err)
	assert.Equal(t, data, b)
}
