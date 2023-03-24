package encoder

import (
	"context"
	"fmt"
	"os"
)

func New(mediaDir, ffmpeg, codec, bitrate string, chunkSizeSeconds int) (*Provider, error) {
	if err := os.MkdirAll(mediaDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to make streams MediaDir %q: %w", mediaDir, err)
	}

	fmt.Printf("ffmpeg settings: mediaDir=%v binary=%v codec=%v bitrate=%v chunkSize=%v\n",
		mediaDir, ffmpeg, codec, bitrate, chunkSizeSeconds)

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
