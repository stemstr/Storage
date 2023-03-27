package service

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"

	db "github.com/stemstr/storage/internal/db/sqlite"
	"github.com/stemstr/storage/internal/encoder"
	blob "github.com/stemstr/storage/internal/storage/blob"
	ls "github.com/stemstr/storage/internal/storage/filesystem"
	"github.com/stemstr/storage/internal/waveform"
)

type Service struct {
	cfg Config
	db  db.DB
	ls  ls.Filesystem
	s3  blob.S3
	enc encoder.Encoder
	viz waveform.Generator
}

func New(cfg Config, db db.DB, ls ls.Filesystem, s3 blob.S3, enc encoder.Encoder, viz waveform.Generator) (*Service, error) {
	return &Service{
		cfg: cfg,
		db:  db,
		ls:  ls,
		s3:  s3,
		enc: enc,
		viz: viz,
	}, nil
}

type NewSampleRequest struct {
	Data     []byte
	Mimetype string
	Pubkey   string
	Sum      string
}

// LocalFilename is the new filename on disk. Sha.ext
func localFilename(sum, mimetype string) string {
	ext := ""
	switch mimetype {
	case "audio/wav", "audio/wave", "audio/x-wav":
		ext = ".wav"
	case "audio/mp3", "audio/mpeg", "audio/x-mpeg-3", "audio/mpeg3":
		ext = ".mp3"
	case "audio/mp4", "audio/m4a":
		ext = ".m4a"
	case "audio/aiff", "audio/x-aiff":
		ext = ".aif"
	}
	return sum + ext
}

// streamFilename is the stream filename on disk. sha without ext
func streamFilename(sum string) string {
	return sum
}

// wavFilename is the WAV filename on disk. sha.wav
func wavFilename(sum string) string {
	return sum + ".wav"
}

type NewSampleResponse struct {
	MediaID  string
	Waveform []int
}

func (s *Service) NewSample(ctx context.Context, r *NewSampleRequest) (*NewSampleResponse, error) {
	// 1. Save original file to disk
	rawMediaPath := filepath.Join(s.cfg.OriginalMediaLocalDir, localFilename(r.Sum, r.Mimetype))
	if err := s.ls.Write(ctx, rawMediaPath, r.Data); err != nil {
		return nil, fmt.Errorf("filesystem.Write: %w", err)
	}

	// 2. Transcoding
	var (
		wg             sync.WaitGroup
		hlsErr, wavErr error
	)
	wg.Add(2)

	streamMediaPath := filepath.Join(s.cfg.StreamMediaLocalDir, streamFilename(r.Sum))
	go func(mimetype, rawMediaPath, streamMediaPath string) {
		defer wg.Done()
		if resp, err := s.enc.HLS(ctx, encoder.EncodeRequest{
			Mimetype:   mimetype,
			InputPath:  rawMediaPath,
			OutputPath: streamMediaPath,
		}); err != nil {
			hlsErr = fmt.Errorf("encoder.HLS: %q: %w", resp.Output, err)
		}
	}(r.Mimetype, rawMediaPath, streamMediaPath)

	wavMediaPath := filepath.Join(s.cfg.WAVMediaLocalDir, wavFilename(r.Sum))
	go func(mimetype, rawMediaPath, wavMediaPath string) {
		defer wg.Done()
		if resp, err := s.enc.WAV(ctx, encoder.EncodeRequest{
			Mimetype:   mimetype,
			InputPath:  rawMediaPath,
			OutputPath: wavMediaPath,
		}); err != nil {
			wavErr = fmt.Errorf("encoder.WAV: %q: %w", resp.Output, err)
		}
	}(r.Mimetype, rawMediaPath, wavMediaPath)

	wg.Wait()

	for _, err := range []error{hlsErr, wavErr} {
		if err != nil {
			return nil, err
		}
	}

	// 3. Generate waveform data
	waveform, err := s.viz.Waveform(ctx, wavMediaPath)
	if err != nil {
		return nil, fmt.Errorf("waveform generate: %w", err)
	}

	// 4. Write to DB
	media, err := s.db.CreateMedia(ctx, db.CreateMediaRequest{
		Size:      int64(len(r.Data)),
		Sum:       r.Sum,
		Mimetype:  r.Mimetype,
		Waveform:  waveform,
		CreatedBy: r.Pubkey,
	})
	if err != nil {
		return nil, fmt.Errorf("db.CreateMedia: %w", err)
	}
	log.Printf("upload: %v created %v\n", media.CreatedBy, media.Mimetype)

	// 5. TODO: Upload to S3?

	return &NewSampleResponse{
		MediaID:  r.Sum,
		Waveform: waveform,
	}, nil
}

func (s *Service) GetSample(ctx context.Context, sum string) (*GetSampleResponse, error) {
	media, err := s.db.GetMedia(ctx, sum)
	if err != nil {
		return nil, fmt.Errorf("db.GetMedia: %w", err)
	}

	var (
		filename     = localFilename(sum, media.Mimetype)
		rawMediaPath = filepath.Join(s.cfg.OriginalMediaLocalDir, filename)
	)

	data, err := s.ls.Read(ctx, rawMediaPath)
	if err != nil {
		return nil, fmt.Errorf("filesystem.Read: %w", err)
	}

	return &GetSampleResponse{
		Media:       media,
		Data:        data,
		Filename:    filename,
		ContentType: media.Mimetype,
	}, nil
}

type GetSampleResponse struct {
	Media       *db.Media
	ContentType string
	Filename    string
	Data        []byte
}
