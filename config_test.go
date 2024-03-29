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
api_base: http://localhost:9000
download_base: http://localhost:9000/download
stream_base: http://localhost:9000/stream
accepted_mimetypes:
  - image/jpg
  - image/png
media_storage_dir: ./files
subscription_options:
  - days: 7
    sats: 1000
  - days: 14
    sats: 2000
`))
	assert.NoError(t, err)

	var cfg Config
	err = cfg.Load(configFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, 9000, cfg.Port)
	assert.Equal(t, "http://localhost:9000", cfg.APIBase)
	assert.Equal(t, "http://localhost:9000/download", cfg.DownloadBase)
	assert.Equal(t, "http://localhost:9000/stream", cfg.StreamBase)
	assert.Equal(t, []string{"image/jpg", "image/png"}, cfg.AcceptedMimetypes)
	assert.Len(t, cfg.SubscriptionOptions, 2)
	assert.Equal(t, 7, cfg.SubscriptionOptions[0].Days)
}

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("PORT", "9000")
	t.Setenv("API_BASE", "http://localhost:9000")
	t.Setenv("DOWNLOAD_BASE", "http://localhost:9000/download")
	t.Setenv("STREAM_BASE", "http://localhost:9000/stream")
	t.Setenv("ACCEPTED_MIMETYPES", "image/jpg,image/png")
	t.Setenv("MEDIA_STORAGE_DIR", "./files")

	var cfg Config
	assert.NoError(t, cfg.LoadFromEnv())
	assert.Equal(t, 9000, cfg.Port)
	assert.Equal(t, "http://localhost:9000", cfg.APIBase)
	assert.Equal(t, "http://localhost:9000/download", cfg.DownloadBase)
	assert.Equal(t, "http://localhost:9000/stream", cfg.StreamBase)
	assert.Equal(t, []string{"image/jpg", "image/png"}, cfg.AcceptedMimetypes)
	assert.Equal(t, "./files", cfg.MediaStorageDir)
}
