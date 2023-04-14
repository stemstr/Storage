package mimes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromFilename(t *testing.T) {
	var tests = []struct {
		filename string
		expected string
	}{
		{"test.aif", "audio/aiff"},
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
