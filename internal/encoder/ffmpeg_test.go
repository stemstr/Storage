package encoder

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFFMPEGEncode(t *testing.T) {
	const ffmpegPath = "/usr/local/bin/ffmpeg"

	// Output encoded files into a gitignored directory and leave them
	// for local testing.
	outputDir := "./testdata/output"
	assert.NoError(t, os.MkdirAll(outputDir, os.FileMode(0777)))

	var tests = []struct {
		fileType   string
		filePath   string
		outputDir  string
		outputName string
	}{
		{"audio/aiff", "./testdata/test.aif", outputDir, "testaif"},
		{"audio/mp3", "./testdata/test.mp3", outputDir, "testmp3"},
		{"audio/mp4", "./testdata/test.m4a", outputDir, "testm4a"},
		{"audio/wave", "./testdata/test.wav", outputDir, "testwav"},
	}

	var (
		ctx = context.Background()
		enc = newFfmpeg(ffmpegPath, encodeOpts{
			ChunkSizeSeconds: 10,
			Codec:            "libmp3lame",
			Bitrate:          "128k",
		})
	)

	for _, tt := range tests {
		t.Run(tt.fileType, func(t *testing.T) {
			_, err := enc.Encode(ctx, EncodeRequest{
				InputPath:  tt.filePath,
				InputType:  tt.fileType,
				OutputDir:  tt.outputDir,
				OutputName: tt.outputName,
			})
			assert.NoError(t, err)
		})
	}
}
