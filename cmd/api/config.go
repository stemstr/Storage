package main

import (
	"os"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
)

const (
	defaultMaxUploadSizeMB        = 2
	defaultStreamFFMPEG           = "ffmpeg"
	defaultStreamChunkSizeSeconds = 10
	defaultStreamCodec            = "libmp3lame"
	defaultStreamBitrate          = "128k"
)

type Config struct {
	Port                      int      `yaml:"port" envconfig:"PORT"`
	APIPath                   string   `yaml:"api_path" envconfig:"API_PATH"`
	StreamBase                string   `yaml:"stream_base" envconfig:"STREAM_BASE"`
	DownloadBase              string   `yaml:"download_base" envconfig:"DOWNLOAD_BASE"`
	MediaStorageDir           string   `yaml:"media_storage_dir" envconfig:"MEDIA_STORAGE_DIR"`
	StreamStorageDir          string   `yaml:"stream_storage_dir" envconfig:"STREAM_STORAGE_DIR"`
	WavStorageDir             string   `yaml:"wav_storage_dir" envconfig:"WAV_STORAGE_DIR"`
	StreamFFMPEG              string   `yaml:"stream_ffmpeg" envconfig:"STREAM_FFMPEG"`
	StreamChunkSizeSeconds    int      `yaml:"stream_chunk_size_seconds" envconfig:"STREAM_CHUNK_SIZE_SECONDS"`
	StreamCodec               string   `yaml:"stream_codec" envconfig:"STREAM_CODEC"`
	StreamBitrate             string   `yaml:"stream_bitrate" envconfig:"STREAM_BITRATE"`
	NostrRelayDBFile          string   `yaml:"nostr_relay_db_file" envconfig:"NOSTR_RELAY_DB_FILE"`
	NostrRelayPort            int      `yaml:"nostr_relay_port" envconfig:"NOSTR_RELAY_PORT"`
	NostrRelayInfoPubkey      string   `yaml:"nostr_relay_info_pubkey" envconfig:"NOSTR_RELAY_INFO_PUBKEY"`
	NostrRelayInfoContact     string   `yaml:"nostr_relay_info_contact" envconfig:"NOSTR_RELAY_INFO_CONTACT"`
	NostrRelayInfoDescription string   `yaml:"nostr_relay_info_description" envconfig:"NOSTR_RELAY_INFO_DESCRIPTION"`
	NostrRelayInfoVersion     string   `yaml:"nostr_relay_info_version" envconfig:"NOSTR_RELAY_INFO_VERSION"`
	DBFile                    string   `yaml:"db_file" envconfig:"DB_FILE"`
	MaxUploadSizeMB           int64    `yaml:"max_upload_size_mb" envconfig:"MAX_UPLOAD_SIZE_MB"`
	AcceptedMimetypes         []string `yaml:"accepted_mimetypes" envconfig:"ACCEPTED_MIMETYPES"`
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
	if c.StreamFFMPEG == "" {
		c.StreamFFMPEG = defaultStreamFFMPEG
	}
	if c.StreamChunkSizeSeconds == 0 {
		c.StreamChunkSizeSeconds = defaultStreamChunkSizeSeconds
	}
	if c.StreamCodec == "" {
		c.StreamCodec = defaultStreamCodec
	}
	if c.StreamBitrate == "" {
		c.StreamBitrate = defaultStreamBitrate
	}
}
