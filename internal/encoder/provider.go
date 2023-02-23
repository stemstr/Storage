package encoder

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
)

const (
	defaultFfmpeg           = "ffmpeg"
	defaultChunkSizeSeconds = 10
	defaultCodec            = "libmp3lame"
	defaultBitrate          = "128k"
)

func New(cfg map[string]string) (*Provider, error) {
	mediaDir := cfg["media_dir"]
	if err := os.MkdirAll(mediaDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to make streams MediaDir %q: %w", mediaDir, err)
	}

	ffmpeg, ok := cfg["ffmpeg"]
	if !ok {
		log.Printf("no stream_config.ffmpeg found. using default %q", defaultFfmpeg)
		ffmpeg = defaultFfmpeg
	}

	chunkSizeSeconds := defaultChunkSizeSeconds
	chunkSizeSecondsStr, ok := cfg["chunk_size_seconds"]
	if ok {
		var err error
		chunkSizeSeconds, err = strconv.Atoi(chunkSizeSecondsStr)
		if err != nil {
			return nil, fmt.Errorf("chunk_size_seconds: %w", err)
		}
	} else {
		log.Printf("no stream_config.chunk_size_seconds found. using default %v", defaultChunkSizeSeconds)
	}
	codec, ok := cfg["codec"]
	if !ok {
		log.Printf("no stream_config.codec found. using default %q", defaultCodec)
		codec = defaultCodec
	}
	bitrate, ok := cfg["bitrate"]
	if !ok {
		log.Printf("no stream_config.bitrate found. using default %q", defaultBitrate)
		bitrate = defaultBitrate
	}

	enc := newFfmpeg(ffmpeg, encodeOpts{
		ChunkSizeSeconds: chunkSizeSeconds,
		Codec:            codec,
		Bitrate:          bitrate,
	})

	return &Provider{
		MediaDir: mediaDir,
		Enc:      enc,
	}, nil
}

type Provider struct {
	MediaDir string
	Enc      Encoder
}

func (p *Provider) EncodeMP3(ctx context.Context, mediaPath, mediaType, name string) (string, error) {
	resp, err := p.Enc.Encode(ctx, EncodeRequest{
		InputPath:  mediaPath,
		InputType:  mediaType,
		OutputDir:  p.MediaDir,
		OutputName: name,
	})

	if err != nil {
		return "", err
	}

	return resp.Path, nil
}
