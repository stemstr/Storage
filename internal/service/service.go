package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"sync"

	db "github.com/stemstr/storage/internal/db/sqlite"
	"github.com/stemstr/storage/internal/encoder"
	"github.com/stemstr/storage/internal/mimes"
	blob "github.com/stemstr/storage/internal/storage/blob"
	ls "github.com/stemstr/storage/internal/storage/filesystem"
	"github.com/stemstr/storage/internal/waveform"
)

var (
	ErrNotFound = errors.New("not found")
)

type Service struct {
	cfg Config
	db  db.DB
	ls  ls.Filesystem
	s3  *blob.S3
	enc encoder.Encoder
	viz waveform.Generator
}

func New(cfg Config, db db.DB, ls ls.Filesystem, s3 *blob.S3, enc encoder.Encoder, viz waveform.Generator) (*Service, error) {
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

type NewSampleResponse struct {
	MediaID  string
	Waveform []int
}

func (s *Service) NewSample(ctx context.Context, r *NewSampleRequest) (*NewSampleResponse, error) {
	var tmpFiles []string

	// 1. Save original file to disk
	rawMediaPath := filepath.Join(s.cfg.OriginalMediaLocalDir, localFilename(r.Sum, r.Mimetype))
	if err := s.ls.Write(ctx, rawMediaPath, r.Data); err != nil {
		return nil, fmt.Errorf("filesystem.Write: %w", err)
	}
	tmpFiles = append(tmpFiles, rawMediaPath)

	// 2. Transcoding
	var (
		wg   sync.WaitGroup
		errs []error
	)
	wg.Add(2)

	// Encode and upload HLS
	streamMediaPath := filepath.Join(s.cfg.StreamMediaLocalDir, streamFilename(r.Sum))
	go func(mimetype, rawMediaPath, streamMediaPath string) {
		defer wg.Done()

		resp, err := s.enc.HLS(ctx, encoder.EncodeRequest{
			Mimetype:   mimetype,
			InputPath:  rawMediaPath,
			OutputPath: streamMediaPath,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("encoder.HLS: %q: %w", resp.Output, err))
		}

		// Upload to S3
		if err := s.uploadHLSToS3(resp); err != nil {
			errs = append(errs, fmt.Errorf("uploadHLSToS3: %w", err))
		}

		tmpFiles = append(tmpFiles, resp.IndexFilepath)
		tmpFiles = append(tmpFiles, resp.SegmentFilepaths...)

	}(r.Mimetype, rawMediaPath, streamMediaPath)

	// Encode and upload WAV
	wavMediaPath := filepath.Join(s.cfg.WAVMediaLocalDir, wavFilename(r.Sum))
	go func(mimetype, rawMediaPath, wavMediaPath string) {
		defer wg.Done()

		resp, err := s.enc.WAV(ctx, encoder.EncodeRequest{
			Mimetype:   mimetype,
			InputPath:  rawMediaPath,
			OutputPath: wavMediaPath,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("encoder.WAV: %q: %w", resp.Output, err))
		}

		// Upload to S3
		if err := s.uploadWAVToS3(resp); err != nil {
			errs = append(errs, fmt.Errorf("uploadWAVToS3: %w", err))
		}

		tmpFiles = append(tmpFiles, resp.Filepath)

	}(r.Mimetype, rawMediaPath, wavMediaPath)

	wg.Wait()

	if len(errs) != 0 {
		return nil, errs[0]
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

	s.ls.Remove(ctx, tmpFiles...)

	return &NewSampleResponse{
		MediaID:  r.Sum,
		Waveform: waveform,
	}, nil
}

func (s *Service) GetSample(ctx context.Context, filename string) (*GetSampleResponse, error) {
	filePath := filepath.Join("download", filename)
	resp, err := s.s3.Get(ctx, filePath)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchKey") {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("s3.Get: %w", err)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	contentType := ""
	if resp.ContentType != nil {
		contentType = *resp.ContentType
	}

	return &GetSampleResponse{
		Data:        data,
		Filename:    filename,
		ContentType: contentType,
	}, nil
}

func (s *Service) GetSampleMetadata(ctx context.Context, sum string) (*GetSampleResponse, error) {
	media, err := s.db.GetMedia(ctx, sum)
	if err != nil {
		return nil, fmt.Errorf("db.GetMedia: %w", err)
	}
	if media == nil {
		return nil, ErrNotFound
	}

	var (
		filename    = wavFilename(sum)
		contentType = mimes.FromFilename(filename)
	)

	return &GetSampleResponse{
		Media:       media,
		Filename:    filename,
		ContentType: contentType,
	}, nil
}

type GetSampleResponse struct {
	Media       *db.Media
	ContentType string
	Filename    string
	Data        []byte
}

func (s *Service) uploadHLSToS3(resp encoder.EncodeHLSResponse) error {
	ctx := context.Background()

	newReq := func(filePath, contentType string) (blob.PutRequest, error) {
		data, err := s.ls.Read(ctx, filePath)
		if err != nil {
			return blob.PutRequest{}, err
		}
		fileSize := int64(len(data))
		filename := filepath.Base(filePath)
		key := filepath.Join("stream", filename)

		return blob.PutRequest{
			Key:           key,
			Body:          bytes.NewReader(data),
			ContentLength: fileSize,
			ContentType:   contentType,
			Metadata: map[string]string{
				"filename": filename,
			},
		}, nil
	}

	var wg sync.WaitGroup
	wg.Add(1 + len(resp.SegmentFilepaths))

	// IndexFile
	var err error
	go func() {
		defer wg.Done()

		var req blob.PutRequest
		req, err = newReq(resp.IndexFilepath, "application/x-mpegURL")
		if err != nil {
			return
		}

		err = s.s3.Put(ctx, req)
	}()

	// Segments
	for _, segmentFilepath := range resp.SegmentFilepaths {
		go func(segmentFilepath string) {
			defer wg.Done()
			var req blob.PutRequest
			req, err = newReq(segmentFilepath, "video/MP2T")
			if err != nil {
				return
			}

			err = s.s3.Put(ctx, req)
		}(segmentFilepath)
	}

	wg.Wait()

	return err
}

func (s *Service) uploadWAVToS3(resp encoder.EncodeWAVResponse) error {
	ctx := context.Background()

	data, err := s.ls.Read(ctx, resp.Filepath)
	if err != nil {
		return err
	}
	fileSize := int64(len(data))
	filename := filepath.Base(resp.Filepath)
	key := filepath.Join("download", filename)

	return s.s3.Put(ctx, blob.PutRequest{
		Key:           key,
		Body:          bytes.NewReader(data),
		ContentLength: fileSize,
		ContentType:   "audio/wave",
		Metadata: map[string]string{
			"filename": filename,
		},
	})
}

// LocalFilename is the new filename on disk. Sha.ext
func localFilename(sum, mimetype string) string {
	ext := mimes.FileExtension(mimetype)
	return sum + ext
}

// streamFilename is the stream filename on disk. sha without ext
func streamFilename(sum string) string {
	return sum
}

// wavFilename is the WAV filename on disk. sha.wav
func wavFilename(sum string) string {
	ext := mimes.FileExtension("audio/wave")
	return sum + ext
}
