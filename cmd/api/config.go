package main

import (
	"os"

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
	// API settings
	Port                   int      `yaml:"port"`
	APIHost                string   `yaml:"api_host"`
	CDNHost                string   `yaml:"cdn_host"`
	S3Bucket               string   `yaml:"s3_bucket"`
	MaxUploadSizeMB        int64    `yaml:"max_upload_size_mb"`
	AcceptedMimetypes      []string `yaml:"accepted_mimetypes"`
	StreamFFMPEG           string   `yaml:"stream_ffmpeg"`
	StreamChunkSizeSeconds int      `yaml:"stream_chunk_size_seconds"`
	StreamCodec            string   `yaml:"stream_codec"`
	StreamBitrate          string   `yaml:"stream_bitrate"`

	// Relay settings
	NostrRelayPort            int    `yaml:"nostr_relay_port"`
	NostrRelayAllowedKinds    []int  `yaml:"nostr_relay_allowed_kinds"`
	NostrRelayInfoPubkey      string `yaml:"nostr_relay_info_pubkey"`
	NostrRelayInfoContact     string `yaml:"nostr_relay_info_contact"`
	NostrRelayInfoDescription string `yaml:"nostr_relay_info_description"`
	NostrRelayInfoVersion     string `yaml:"nostr_relay_info_version"`
	NostrRelayDatabaseURL     string `yaml:"nostr_relay_database_url"`
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
