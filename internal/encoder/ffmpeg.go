package encoder

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// newFfmpeg returns a new ffmpeg Encoder
func newFfmpeg(binPath string, opts encodeOpts) Encoder {
	return &ffmpegEncoder{
		bin:  binPath,
		opts: opts,
	}
}

// https://trac.ffmpeg.org/wiki/Encode/HighQualityAudio
type encodeOpts struct {
	ChunkSizeSeconds int
	Codec            string
	Bitrate          string
}

type ffmpegEncoder struct {
	bin  string
	opts encodeOpts
}

// Encode encodes the provide audio file into HLS chunked mp3s.
func (e *ffmpegEncoder) Encode(ctx context.Context, req EncodeRequest) (EncodeResponse, error) {
	var args []string
	switch strings.ToLower(req.InputType) {
	case
		// .wav
		"audio/wav", "audio/wave", "audio/x-wav",
		// .mp3
		"audio/mp3", "audio/mpeg", "audio/x-mpeg-3", "audio/mpeg3",
		// .m4a
		"audio/mp4", "audio/m4a",
		// .aif
		"audio/aiff", "audio/x-aiff":
		args = defaultArgs(e.opts, req.InputPath, req.OutputDir, req.OutputName)
	default:
		return EncodeResponse{}, ErrUnsupportedType
	}

	cmd := exec.Command(e.bin, args...)
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		log.Printf("encode failure: %v\n cmd=%q", err, cmd.String())
		return EncodeResponse{}, err
	}

	return EncodeResponse{
		Output: out.String(),
		Path:   filepath.Join(req.OutputDir, req.OutputName+".m3u8"),
	}, nil
}

func defaultArgs(opts encodeOpts, inputPath, outputDir, name string) []string {
	var (
		outputIndex      = fmt.Sprintf("%s.m3u8", name)
		outputChunk      = fmt.Sprintf("%s%%03d.ts", name)
		outputIndexPath  = filepath.Join(outputDir, outputIndex)
		outputChunksPath = filepath.Join(outputDir, outputChunk)
	)

	return []string{
		"-i", inputPath,
		"-b:a", opts.Bitrate,
		"-c:a", opts.Codec,
		"-f", "segment",
		"-map", "a", // Strip artwork
		"-segment_time", strconv.Itoa(opts.ChunkSizeSeconds),
		"-segment_list", outputIndexPath,
		"-segment_format", "mpegts",
		outputChunksPath,
	}
}
