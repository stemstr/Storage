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
media_path: http://localhost:9000
accepted_mimetypes:
  - image/jpg
  - image/png
storage_type: filesystem
storage_config:
  media_dir: ./files
`))
	assert.NoError(t, err)

	var cfg Config
	err = cfg.Load(configFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, 9000, cfg.Port)
	assert.Equal(t, "http://localhost:9000/upload", cfg.APIPath)
	assert.Equal(t, "http://localhost:9000", cfg.MediaPath)
	assert.Equal(t, []string{"image/jpg", "image/png"}, cfg.AcceptedMimetypes)
	assert.Equal(t, "filesystem", cfg.StorageType)
}

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("PORT", "9000")
	t.Setenv("API_PATH", "http://localhost:9000/upload")
	t.Setenv("MEDIA_PATH", "http://localhost:9000")
	t.Setenv("ACCEPTED_MIMETYPES", "image/jpg,image/png")
	t.Setenv("STORAGE_TYPE", "filesystem")
	t.Setenv("STORAGE_CONFIG", "media_dir:./files")

	var cfg Config
	assert.NoError(t, cfg.LoadFromEnv())
	assert.Equal(t, 9000, cfg.Port)
	assert.Equal(t, "http://localhost:9000/upload", cfg.APIPath)
	assert.Equal(t, "http://localhost:9000", cfg.MediaPath)
	assert.Equal(t, []string{"image/jpg", "image/png"}, cfg.AcceptedMimetypes)
	assert.Equal(t, "filesystem", cfg.StorageType)
}
