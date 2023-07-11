package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/stemstr/storage/internal/encoder"
	"github.com/stemstr/storage/internal/service"
	blob "github.com/stemstr/storage/internal/storage/blob"
	ls "github.com/stemstr/storage/internal/storage/filesystem"
	"github.com/stemstr/storage/internal/waveform"
)

var (
	commit    string
	buildDate string
)

func main() {
	ctx := context.Background()

	configPath := flag.String("config", "config.yaml", "location of config file. If non is specified config will be loaded from the environment")
	flag.Parse()

	log.Printf("build info: commit: %v date: %v\n", commit, buildDate)
	log.Printf("loading config from %s\n", *configPath)

	var cfg Config
	if err := cfg.Load(*configPath); err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// Relay setup
	relay, err := newRelay(relayConfig{
		Port:             cfg.NostrRelayPort,
		DatabaseURL:      cfg.NostrRelayDatabaseURL,
		AllowedKinds:     cfg.NostrRelayAllowedKinds,
		Nip11Pubkey:      cfg.NostrRelayInfoPubkey,
		Nip11Contact:     cfg.NostrRelayInfoContact,
		Nip11Description: cfg.NostrRelayInfoDescription,
		Nip11Version:     cfg.NostrRelayInfoVersion,
	})
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	go func() {
		if err := relay.Start(); err != nil {
			log.Printf("relay err: %v\n", err)
			os.Exit(1)
		}
	}()

	// Encoder setup
	enc := encoder.New(cfg.StreamFFMPEG, encoder.EncodeOpts{
		ChunkSizeSeconds: cfg.StreamChunkSizeSeconds,
		Codec:            cfg.StreamCodec,
		Bitrate:          cfg.StreamBitrate,
	})

	s3, err := blob.New(ctx, cfg.S3Bucket)
	if err != nil {
		log.Printf("s3 err: %v\n", err)
		os.Exit(1)
	}

	// Service setup
	var (
		ls  = ls.New()
		viz = waveform.New(enc)
	)

	svc, err := service.New(ls, s3, enc, viz)
	if err != nil {
		log.Printf("service err: %v\n", err)
		os.Exit(1)
	}

	h := handlers{
		config: cfg,
		svc:    svc,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Content-Disposition", "Link", "X-Download-Filename"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/download/{filename}", h.handleDownloadMedia)
	r.Get("/stream/{filename}", h.handleGetStream)
	r.Post("/upload", h.handleUpload)
	r.Get("/debug/stream", h.handleDebugStream)

	port := fmt.Sprintf(":%d", cfg.Port)

	log.Printf("api listening on %v\n", port)
	http.ListenAndServe(port, r)
}

func createDirIfNotExists(path string) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}
