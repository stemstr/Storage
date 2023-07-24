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
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/stemstr/storage/internal/encoder"
	"github.com/stemstr/storage/internal/service"
	blob "github.com/stemstr/storage/internal/storage/blob"
	ls "github.com/stemstr/storage/internal/storage/filesystem"
	"github.com/stemstr/storage/internal/subscription"
	"github.com/stemstr/storage/internal/subscription/ln/nodeless"
	"github.com/stemstr/storage/internal/subscription/repo/pg"
	"github.com/stemstr/storage/internal/waveform"
)

var (
	commit    string
	buildDate string
)

func main() {
	ctx := context.Background()

	configPath := flag.String("config", "", "location of config file. If non is specified config will be loaded from the environment")
	flag.Parse()

	log.Printf("build info: commit: %v date: %v\n", commit, buildDate)

	var (
		cfg Config
		err error
	)
	if *configPath != "" {
		log.Printf("loading config from file %q\n", *configPath)
		err = cfg.Load(*configPath)
	} else {
		log.Println("loading config from env")
		err = cfg.LoadFromEnv()
	}
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// Ensure the local directories exist
	if err := createDirIfNotExists(cfg.MediaStorageDir); err != nil {
		log.Println(err)
		os.Exit(1)
	}
	if err := createDirIfNotExists(cfg.StreamStorageDir); err != nil {
		log.Println(err)
		os.Exit(1)
	}
	if err := createDirIfNotExists(cfg.WavStorageDir); err != nil {
		log.Println(err)
		os.Exit(1)
	}

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

	// Subscriptions setup
	lnProvider, err := nodeless.New(cfg.NodelessAPIKey, cfg.NodelessStoreID, cfg.NodelessTestnet)
	if err != nil {
		log.Printf("nodeless err: %v\n", err)
		os.Exit(1)
	}
	subRepo, err := pg.New("fixme")
	if err != nil {
		log.Printf("subRepo err: %v\n", err)
		os.Exit(1)
	}
	subService, err := subscription.New(subRepo, lnProvider)
	if err != nil {
		log.Printf("subRepo err: %v\n", err)
		os.Exit(1)
	}

	// Service setup
	var (
		svcConfig = service.Config{
			OriginalMediaLocalDir: cfg.MediaStorageDir,
			StreamMediaLocalDir:   cfg.StreamStorageDir,
			WAVMediaLocalDir:      cfg.WavStorageDir,
		}
		ls  = ls.New()
		viz = waveform.New(enc)
	)

	svc, err := service.New(svcConfig, ls, s3, enc, viz)
	if err != nil {
		log.Printf("service err: %v\n", err)
		os.Exit(1)
	}

	h := handlers{
		config: cfg,
		svc:    svc,
		subs:   subService,
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
	r.Use(metricsMiddleware)

	r.Get("/download/{filename}", h.handleDownloadMedia)
	r.Get("/stream/{filename}", h.handleGetStream)
	r.Get("/subscription/{pubkey}", h.handleGetSubscription)
	r.Post("/subscription/{pubkey}", h.handleCreateSubscription)
	r.Post("/upload", h.handleUpload)
	r.Method(http.MethodGet, "/metrics", promhttp.Handler())
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
