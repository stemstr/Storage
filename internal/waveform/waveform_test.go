package waveform

import (
	"context"
	"testing"

	"github.com/stemstr/storage/internal/encoder"
	"github.com/stretchr/testify/assert"
)

func TestWaveformGenerator(t *testing.T) {
	const (
		ffmpegPath = "/usr/local/bin/ffmpeg"
		audioFile  = "../encoder/testdata/test.wav"
	)

	enc := encoder.New(ffmpegPath, encoder.EncodeOpts{
		ChunkSizeSeconds: 10,
		Codec:            "libmp3lame",
		Bitrate:          "128k",
	})

	data, err := New(enc).Waveform(context.Background(), audioFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
}
