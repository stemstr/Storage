package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfigFromFile(t *testing.T) {
	configFile, err := ioutil.TempFile("", "config.*.yml")
	assert.NoError(t, err)

	defer os.Remove(configFile.Name())

	_, err = configFile.Write([]byte(`
port: 9000
api_path: http://localhost:9000/upload
download_base: http://localhost:9000/download
stream_base: http://localhost:9000/stream
accepted_mimetypes:
  - image/jpg
  - image/png
media_storage_type: filesystem
media_storage_dir: ./files
`))
	assert.NoError(t, err)

	var cfg Config
	err = cfg.Load(configFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, 9000, cfg.Port)
	assert.Equal(t, "http://localhost:9000/upload", cfg.APIPath)
	assert.Equal(t, "http://localhost:9000/download", cfg.DownloadBase)
	assert.Equal(t, "http://localhost:9000/stream", cfg.StreamBase)
	assert.Equal(t, []string{"image/jpg", "image/png"}, cfg.AcceptedMimetypes)
	assert.Equal(t, "filesystem", cfg.MediaStorageType)
}

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("PORT", "9000")
	t.Setenv("API_PATH", "http://localhost:9000/upload")
	t.Setenv("DOWNLOAD_BASE", "http://localhost:9000/download")
	t.Setenv("STREAM_BASE", "http://localhost:9000/stream")
	t.Setenv("ACCEPTED_MIMETYPES", "image/jpg,image/png")
	t.Setenv("MEDIA_STORAGE_TYPE", "filesystem")
	t.Setenv("MEDIA_STORAGE_DIR", "./files")

	var cfg Config
	assert.NoError(t, cfg.LoadFromEnv())
	assert.Equal(t, 9000, cfg.Port)
	assert.Equal(t, "http://localhost:9000/upload", cfg.APIPath)
	assert.Equal(t, "http://localhost:9000/download", cfg.DownloadBase)
	assert.Equal(t, "http://localhost:9000/stream", cfg.StreamBase)
	assert.Equal(t, []string{"image/jpg", "image/png"}, cfg.AcceptedMimetypes)
	assert.Equal(t, "filesystem", cfg.MediaStorageType)
	assert.Equal(t, "./files", cfg.MediaStorageDir)
}
