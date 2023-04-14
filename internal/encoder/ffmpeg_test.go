package encoder

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const ffmpegPath = "ffmpeg"

func TestHLS(t *testing.T) {
	// Output encoded files into a gitignored directory and leave them
	// for local testing.
	outputDir := "./testdata/output"
	assert.NoError(t, os.MkdirAll(outputDir, os.FileMode(0777)))

	var tests = []struct {
		mimeType   string
		inputPath  string
		outputPath string
	}{
		{"audio/aiff", "./testdata/test.aif", filepath.Join(outputDir, "test.aif")},
		{"audio/flac", "./testdata/test.flac", filepath.Join(outputDir, "test.flac")},
		{"audio/mp3", "./testdata/test.mp3", filepath.Join(outputDir, "test.mp3")},
		{"audio/mp4", "./testdata/test.m4a", filepath.Join(outputDir, "test.m4a")},
		{"audio/wave", "./testdata/test.wav", filepath.Join(outputDir, "test.wav")},
		{"audio/ogg", "./testdata/test.ogg", filepath.Join(outputDir, "test.ogg")},
	}

	var (
		ctx = context.Background()
		enc = New(ffmpegPath, EncodeOpts{
			ChunkSizeSeconds: 10,
			Codec:            "libmp3lame",
			Bitrate:          "128k",
		})
	)

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			_, err := enc.HLS(ctx, EncodeRequest{
				Mimetype:   tt.mimeType,
				InputPath:  tt.inputPath,
				OutputPath: tt.outputPath,
			})
			assert.NoError(t, err)
		})
	}
}

func TestWAV(t *testing.T) {
	// Output encoded files into a gitignored directory and leave them
	// for local testing.
	outputDir := "./testdata/output"
	assert.NoError(t, os.MkdirAll(outputDir, os.FileMode(0777)))

	var tests = []struct {
		mimeType   string
		inputPath  string
		outputPath string
	}{
		{"audio/aiff", "./testdata/test.aif", filepath.Join(outputDir, "test.wav")},
		{"audio/flac", "./testdata/test.flac", filepath.Join(outputDir, "test.wav")},
		{"audio/mp3", "./testdata/test.mp3", filepath.Join(outputDir, "test.wav")},
		{"audio/mp4", "./testdata/test.m4a", filepath.Join(outputDir, "test.wav")},
		{"audio/wave", "./testdata/test.wav", filepath.Join(outputDir, "test.wav")},
		{"audio/ogg", "./testdata/test.ogg", filepath.Join(outputDir, "test.wav")},
	}

	var (
		ctx = context.Background()
		enc = New(ffmpegPath, EncodeOpts{
			ChunkSizeSeconds: 10,
			Codec:            "libmp3lame",
			Bitrate:          "128k",
		})
	)

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			_, err := enc.WAV(ctx, EncodeRequest{
				Mimetype:   tt.mimeType,
				InputPath:  tt.inputPath,
				OutputPath: tt.outputPath,
			})
			assert.NoError(t, err)
			assert.NoError(t, os.Remove(tt.outputPath))
		})
	}
}
