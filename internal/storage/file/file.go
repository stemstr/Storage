package file

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
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

	if err := os.MkdirAll(mediaDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to make MediaDir: %w", err)
	}

	return &Provider{
		MediaDir: mediaDir,
	}, nil
}

type Provider struct {
	MediaDir string
}

func (p *Provider) Save(ctx context.Context, src io.Reader, sum string) error {
	targetDir := filepath.Join(p.MediaDir, sum)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to make file dir: %w", err)
		}
	} else {
		// We already have this file.
		return nil
	}

	fullPath := filepath.Join(targetDir, ondiskFilename)
	target, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer target.Close()

	if _, err = io.Copy(target, src); err != nil {
		return err
	}

	return nil
}

func (p *Provider) Get(ctx context.Context, sum string) (io.Reader, error) {
	fileDir := filepath.Join(p.MediaDir, sum, ondiskFilename)
	return os.Open(fileDir)
}
