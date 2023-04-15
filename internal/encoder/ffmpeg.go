package encoder

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// newFfmpeg returns a new ffmpeg Encoder
func New(binPath string, opts EncodeOpts) Encoder {
	return &ffmpegEncoder{
		bin:  binPath,
		opts: opts,
	}
}

// https://trac.ffmpeg.org/wiki/Encode/HighQualityAudio
type EncodeOpts struct {
	ChunkSizeSeconds int
	Codec            string
	Bitrate          string
}

type ffmpegEncoder struct {
	bin  string
	opts EncodeOpts
}

// HLS encodes the provided audio file into HLS chunked mp3s.
func (e *ffmpegEncoder) HLS(ctx context.Context, req EncodeRequest) (EncodeResponse, error) {
	// Skip if already exists
	indexFile := hlsIndexPath(req.OutputPath)
	if _, err := os.Stat(indexFile); err == nil {
		log.Printf("HLS: already exists: %v\n", indexFile)
		return EncodeResponse{}, nil
	}

	args := defaultHLSArgs(e.opts, req.InputPath, req.OutputPath)

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
	}, nil
}

// WAV encodes the provided audio file as a WAV.
func (e *ffmpegEncoder) WAV(ctx context.Context, req EncodeRequest) (EncodeResponse, error) {
	// Skip if already exists
	if _, err := os.Stat(req.OutputPath); err == nil {
		log.Printf("WAV: already exists: %v\n", req.OutputPath)
		return EncodeResponse{}, nil
	}

	var args []string
	switch strings.ToLower(req.Mimetype) {
	case "audio/wav", "audio/wave", "audio/x-wav":
		if err := copyFile(req.InputPath, req.OutputPath); err != nil {
			return EncodeResponse{}, err
		}
		return EncodeResponse{}, nil
	default:
		args = defaultWAVArgs(e.opts, req.InputPath, req.OutputPath)
	}

	cmd := exec.Command(e.bin, args...)
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		log.Printf("encode failure: %v\n cmd=%q", err, cmd.String())
		return EncodeResponse{Output: out.String()}, err
	}

	return EncodeResponse{
		Output: out.String(),
	}, nil
}

func hlsIndexPath(outputPath string) string {
	return fmt.Sprintf("%s.m3u8", outputPath)
}

func hlsChunksPath(outputPath string) string {
	return fmt.Sprintf("%s%%03d.ts", outputPath)
}

func defaultHLSArgs(opts EncodeOpts, inputPath, outputPath string) []string {
	var (
		outputIndexPath  = hlsIndexPath(outputPath)
		outputChunksPath = hlsChunksPath(outputPath)
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

func defaultWAVArgs(opts EncodeOpts, inputPath, outputPath string) []string {
	// ffmpeg -i test.aif -acodec pcm_s16le -ac 2 -ar 44100 testaif.wav

	return []string{
		"-i", inputPath,
		"-acodec", "pcm_s16le",
		"-ac", "2",
		"-ar", "44100",
		outputPath,
	}
}

func copyFile(src, dst string) error {
	stat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !stat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
