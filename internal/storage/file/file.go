package file

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/stemstr/storage/internal/storage"
)

const (
	defaultMediaDir = "./files"
	ondiskFilename  = "data"
)

func New(cfg map[string]string) (*Provider, error) {
	mediaDir, ok := cfg["media_dir"]
	if !ok {
		log.Printf("no storage_config.media_dir found. using default %q", defaultMediaDir)
		mediaDir = defaultMediaDir
	}

	return &Provider{
		MediaDir: mediaDir,
	}, nil
}

type Provider struct {
	MediaDir string
}

func (p *Provider) Save(ctx context.Context, src io.Reader, opts storage.Options) (string, error) {
	if opts.Sha256 == "" {
		return "", fmt.Errorf("Must provide sha256 of file content")
	}
	if opts.Filename == "" {
		// TODO: Spec says filenames are optional and should be "_" if not provided
		return "", fmt.Errorf("Must provide filename")
	}

	// mediaPath is the filepath relative to the config.MediaPath.
	mediaPath := filepath.Join(opts.Sha256, opts.Filename)

	if err := os.MkdirAll(p.MediaDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to make MediaDir: %w", err)
	}

	targetDir := filepath.Join(p.MediaDir, opts.Sha256)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
			return "", fmt.Errorf("failed to make file dir: %w", err)
		}
	} else {
		// We already have this file.
		return mediaPath, nil
	}

	fullPath := filepath.Join(targetDir, ondiskFilename)
	target, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		return "", err
	}
	defer target.Close()

	if _, err = io.Copy(target, src); err != nil {
		return "", err
	}

	return mediaPath, nil
}

func (p *Provider) Get(ctx context.Context, sum, name string) (io.Reader, error) {
	fileDir := filepath.Join(p.MediaDir, sum, ondiskFilename)
	return os.Open(fileDir)
}
