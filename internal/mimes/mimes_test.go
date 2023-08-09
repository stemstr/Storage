package mimes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileExtension(t *testing.T) {
	var tests = []struct {
		mime     string
		expected string
	}{
		{"audio/aiff", ".aif"},
		{"audio/x-aiff", ".aif"},
		{"audio/flac", ".flac"},
		{"audio/mp3", ".mp3"},
		{"audio/mpeg3", ".mp3"},
		{"audio/x-mpeg-3", ".mp3"},
		{"audio/mpeg", ".mp3"},
		{"audio/mp4", ".m4a"},
		{"audio/ogg", ".ogg"},
		{"audio/wave", ".wav"},
		{"audio/wav", ".wav"},
		{"audio/x-wav", ".wav"},
		{"unsupported", ""},
	}

	for _, tt := range tests {
		t.Run(tt.mime, func(t *testing.T) {
			ext := FileExtension(tt.mime)
			assert.Equal(t, tt.expected, ext)
		})
	}
}

func TestFromFilename(t *testing.T) {
	var tests = []struct {
		filename string
		expected string
	}{
		{"test.aif", "audio/aiff"},
		{"test.aiff", "audio/aiff"},
		{"test.flac", "audio/flac"},
		{"test.mp3", "audio/mp3"},
		{"test.m4a", "audio/mp4"},
		{"test.wav", "audio/wave"},
		{"test.ogg", "audio/ogg"},
		{"test.wtf", ""},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			resp := FromFilename(tt.filename)
			assert.Equal(t, tt.expected, resp)
		})
	}
}
