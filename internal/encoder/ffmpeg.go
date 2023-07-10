package encoder

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
func (e *ffmpegEncoder) HLS(ctx context.Context, req EncodeRequest) (EncodeHLSResponse, error) {
	args := defaultHLSArgs(e.opts, req.InputPath, req.OutputPath)

	cmd := exec.Command(e.bin, args...)
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		log.Printf("encode failure: %v\n cmd=%q", err, cmd.String())
		return EncodeHLSResponse{}, err
	}

	// PERF: req.OutputPath is a single directory shared for all uploads. This
	// method will scale with O(n) complexity while media is stored on the same
	// filesystem. This will not be a problem after filestorage is offloaded to
	// S3 because the local files will be deleted after successful upload.
	segments, err := segmentFilepaths(req.OutputPath)
	if err != nil {
		log.Printf("segmentFilepaths: %v\n outputPath=%q", err, req.OutputPath)
		return EncodeHLSResponse{}, err
	}

	return EncodeHLSResponse{
		Output:           out.String(),
		IndexFilepath:    hlsIndexPath(req.OutputPath),
		SegmentFilepaths: segments,
	}, nil
}

// WAV encodes the provided audio file as a WAV.
func (e *ffmpegEncoder) WAV(ctx context.Context, req EncodeRequest) (EncodeWAVResponse, error) {
	var args []string
	switch strings.ToLower(req.Mimetype) {
	case "audio/wav", "audio/wave", "audio/x-wav":
		if err := copyFile(req.InputPath, req.OutputPath); err != nil {
			return EncodeWAVResponse{}, err
		}
		return EncodeWAVResponse{}, nil
	default:
		args = defaultWAVArgs(e.opts, req.InputPath, req.OutputPath)
	}

	cmd := exec.Command(e.bin, args...)
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		log.Printf("encode failure: %v\n cmd=%q", err, cmd.String())
		return EncodeWAVResponse{Output: out.String()}, err
	}

	return EncodeWAVResponse{
		Output:   out.String(),
		Filepath: req.OutputPath,
	}, nil
}

func hlsIndexPath(outputPath string) string {
	return fmt.Sprintf("%s.m3u8", outputPath)
}

func hlsChunksPath(outputPath string) string {
	return fmt.Sprintf("%s%%03d.ts", outputPath)
}

func segmentFilepaths(outputPath string) ([]string, error) {
	var (
		indexFile  = hlsIndexPath(outputPath)
		streamsDir = filepath.Dir(indexFile)
	)

	// Find stream chunks for this index file in the same directory
	files, err := ioutil.ReadDir(streamsDir)
	if err != nil {
		return nil, err
	}

	var (
		indexFileBase = filepath.Base(indexFile)
		relOutput     = filepath.Base(outputPath)
		segments      []string
	)
	for _, f := range files {
		fn := f.Name()
		if !strings.EqualFold(fn, indexFileBase) && strings.HasPrefix(fn, relOutput) {
			segments = append(segments, filepath.Join(streamsDir, fn))
		}
	}

	return segments, nil
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
