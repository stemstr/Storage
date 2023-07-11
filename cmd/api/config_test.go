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
api_host: http://localhost:9000
accepted_mimetypes:
  - image/jpg
  - image/png
`))
	assert.NoError(t, err)

	var cfg Config
	err = cfg.Load(configFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, 9000, cfg.Port)
	assert.Equal(t, "http://localhost:9000", cfg.APIHost)
	assert.Equal(t, []string{"image/jpg", "image/png"}, cfg.AcceptedMimetypes)
}
