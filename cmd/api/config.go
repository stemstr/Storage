package main

import (
	"log"
	"os"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
)

const (
	defaultMaxUploadSizeMB = 2
	defaultStreamsMediaDir = "./streams"
)

type Config struct {
	APIPath           string            `yaml:"api_path" envconfig:"API_PATH"`
	DownloadBase      string            `yaml:"download_base" envconfig:"DOWNLOAD_BASE"`
	StreamBase        string            `yaml:"stream_base" envconfig:"STREAM_BASE"`
	AcceptedMimetypes []string          `yaml:"accepted_mimetypes" envconfig:"ACCEPTED_MIMETYPES"`
	Port              int               `yaml:"port" envconfig:"PORT"`
	StorageType       string            `yaml:"storage_type" envconfig:"STORAGE_TYPE"`
	StorageConfig     map[string]string `yaml:"storage_config" envconfig:"STORAGE_CONFIG"`
	StreamConfig      map[string]string `yaml:"stream_config" envconfig:"STREAM_CONFIG"`
	MaxUploadSizeMB   int64             `yaml:"max_upload_size_mb" envconfig:"MAX_UPLOAD_SIZE_MB"`
	NostrRelayDBFile  string            `yaml:"nostr_relay_db_file" envconfig:"NOSTR_RELAY_DB_FILE"`
	NostrRelayPort    int               `yaml:"nostr_relay_port" envconfig:"NOSTR_RELAY_PORT"`
	DBFile            string            `yaml:"db_file" envconfig:"DB_FILE"`
}

// Load Config from a yaml file at path.
func (c *Config) Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(c); err != nil {
		return err
	}

	c.applyDefaults()
	return nil
}

// Load Config from the environment.
func (c *Config) LoadFromEnv() error {
	if err := envconfig.Process("", c); err != nil {
		return err
	}

	c.applyDefaults()
	return nil
}

func (c *Config) applyDefaults() {
	if c.MaxUploadSizeMB == 0 {
		c.MaxUploadSizeMB = defaultMaxUploadSizeMB
	}

	if _, ok := c.StreamConfig["media_dir"]; !ok {
		log.Printf("no stream_config.media_dir found. using default %q", defaultStreamsMediaDir)
		if c.StreamConfig == nil {
			c.StreamConfig = map[string]string{}
		}
		c.StreamConfig["media_dir"] = defaultStreamsMediaDir
	}
}
