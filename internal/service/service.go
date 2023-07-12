package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

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
	ls  ls.Filesystem
	s3  *blob.S3
	enc encoder.Encoder
	viz waveform.Generator
}

func New(ls ls.Filesystem, s3 *blob.S3, enc encoder.Encoder, viz waveform.Generator) (*Service, error) {
	return &Service{
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

func (r NewSampleResponse) StreamURL(host string) string {
	u, _ := url.JoinPath(host, "stream", r.MediaID+".m3u8")
	return u

}
func (r NewSampleResponse) DownloadURL(host string) string {
	u, _ := url.JoinPath(host, "download", r.MediaID+".wav")
	return u
}

func (s *Service) NewSample(ctx context.Context, r *NewSampleRequest) (*NewSampleResponse, error) {
	tmpDir := filepath.Join(os.TempDir(), r.Sum)
	if err := os.MkdirAll(tmpDir, 0766); err != nil {
		return nil, fmt.Errorf("os.MkdirAll %q: %w", tmpDir, err)
	}
	defer os.RemoveAll(tmpDir)

	// 1. Save source file to disk
	sourceMediaPath := filepath.Join(tmpDir, localFilename(r.Sum, r.Mimetype))
	if err := s.ls.Write(ctx, sourceMediaPath, r.Data); err != nil {
		return nil, fmt.Errorf("filesystem.Write: %w", err)
	}

	// 2. Transcoding
	var (
		wg   sync.WaitGroup
		errs []error
	)
	wg.Add(2)

	// Encode and upload HLS
	streamMediaPath := filepath.Join(tmpDir, streamFilename(r.Sum))
	go func(mimetype, sourceMediaPath, streamMediaPath string) {
		defer wg.Done()

		resp, err := s.enc.HLS(ctx, encoder.EncodeRequest{
			Mimetype:   mimetype,
			InputPath:  sourceMediaPath,
			OutputPath: streamMediaPath,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("encoder.HLS: %q: %w", resp.Output, err))
		}

		log.Printf("hls resp: %#v\n", resp)

		// Upload to S3
		if err := s.uploadHLSToS3(resp); err != nil {
			errs = append(errs, fmt.Errorf("uploadHLSToS3: %w", err))
		}

	}(r.Mimetype, sourceMediaPath, streamMediaPath)

	// Encode and upload WAV
	wavMediaPath := filepath.Join(tmpDir, wavFilename(r.Sum))
	go func(mimetype, sourceMediaPath, wavMediaPath string) {
		defer wg.Done()

		resp, err := s.enc.WAV(ctx, encoder.EncodeRequest{
			Mimetype:   mimetype,
			InputPath:  sourceMediaPath,
			OutputPath: wavMediaPath,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("encoder.WAV: %q: %w", resp.Output, err))
		}

		log.Printf("wav resp: %#v\n", resp)

		// Upload to S3
		if err := s.uploadWAVToS3(resp); err != nil {
			errs = append(errs, fmt.Errorf("uploadWAVToS3: %w", err))
		}

	}(r.Mimetype, sourceMediaPath, wavMediaPath)

	wg.Wait()

	if len(errs) != 0 {
		return nil, errs[0]
	}

	// 3. Generate waveform data
	waveform, err := s.viz.Waveform(ctx, wavMediaPath)
	if err != nil {
		return nil, fmt.Errorf("waveform generate: %w", err)
	}

	log.Printf("upload: %v created %v\n", r.Pubkey, r.Mimetype)

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

type GetSampleResponse struct {
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
