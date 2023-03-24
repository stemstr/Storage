package main

import (
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
	"github.com/stemstr/storage/internal/storage/file"
)

func main() {
	configPath := flag.String("config", "", "location of config file. If non is specified config will be loaded from the environment")
	flag.Parse()

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

	// Relay setup
	relay := newRelay(cfg.NostrRelayDBFile, cfg.NostrRelayPort)
	go func() {
		if err := relay.Start(); err != nil {
			log.Printf("relay err: %v\n", err)
			os.Exit(1)
		}
	}()

	// Media storage setup
	var store storageProvider
	switch cfg.MediaStorageType {
	default:
		log.Printf("missing or unknown storage_type. using 'filesystem'")
		fallthrough
	case "filesystem":
		store, err = file.New(cfg.MediaStorageDir)
		if err != nil {
			log.Printf("storage err: %v\n", err)
			os.Exit(1)
		}
	}

	// Encoder setup
	enc, err := encoder.New(cfg.StreamStorageDir, cfg.StreamFFMPEG,
		cfg.StreamCodec, cfg.StreamBitrate, cfg.StreamChunkSizeSeconds)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

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

	h := handlers{
		Config:  cfg,
		Store:   store,
		Encoder: enc,
		Relay:   relay,
		DB:      db,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(cors.AllowAll().Handler)
	r.Use(metricsMiddleware)

	r.Post("/beta-signup", h.handleBetaSignup)
	r.Get("/download/{sum}", h.handleDownloadMedia)
	r.Post("/event", h.handlePostEvent)
	r.Post("/upload/quote", h.handleGetQuote)
	r.Post("/upload", h.handleUploadMedia)
	r.Method(http.MethodGet, "/metrics", promhttp.Handler())
	r.Get("/debug/stream", h.handleDebugStream)

	fileServer(r, streamRoute, http.Dir(cfg.StreamStorageDir))

	port := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("listening on %v\n", port)

	http.ListenAndServe(port, r)
}
