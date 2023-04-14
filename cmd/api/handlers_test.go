package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectContentType(t *testing.T) {
	var tests = []struct {
		filePath string
		expected string
	}{
		{"../../internal/encoder/testdata/test.aif", "audio/aiff"},
		{"../../internal/encoder/testdata/test.mp3", "audio/mp3"},
		{"../../internal/encoder/testdata/test.m4a", "audio/mp4"},
		{"../../internal/encoder/testdata/test.wav", "audio/wave"},
		{"../../internal/encoder/testdata/test.ogg", "audio/ogg"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			data, err := os.ReadFile(tt.filePath)
			assert.NoError(t, err)

			fileName := filepath.Base(tt.filePath)
			contentType := detectContentType(data, &fileName)
			assert.Equal(t, tt.expected, contentType)
		})
	}
}
