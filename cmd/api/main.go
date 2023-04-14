package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/stemstr/storage/internal/db/sqlite"
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

	// Relay setup
	relay := newRelay(
		cfg.NostrRelayPort,
		cfg.NostrRelayDBFile,
		cfg.NostrRelayInfoPubkey,
		cfg.NostrRelayInfoContact,
		cfg.NostrRelayInfoDescription,
		cfg.NostrRelayInfoVersion,
	)

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

	streamURL, err := url.Parse(cfg.StreamBase)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	streamRoute := streamURL.Path

	// DB setup
	db, err := sqlite.New(cfg.DBFile)
	if err != nil {
		log.Printf("db err: %v\n", err)
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
		s3  = blob.New()
		viz = waveform.New(enc)
	)

	svc, err := service.New(svcConfig, db, ls, s3, enc, viz)
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
	r.Use(metricsMiddleware)

	r.Get("/download/{sum}", h.handleDownloadMedia)
	r.Get("/metadata/{sum}", h.handleGetMetadata)
	r.Post("/upload", h.handleUpload)
	r.Method(http.MethodGet, "/metrics", promhttp.Handler())
	r.Get("/debug/stream", h.handleDebugStream)

	fileServer(r, streamRoute, http.Dir(cfg.StreamStorageDir))

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
