package main

import (
	"context"
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
	ctx := context.Background()
	relay, err := connectToRelay(ctx, cfg.NostrRelay)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// Storage setup
	var store storageProvider
	switch cfg.StorageType {
	default:
		log.Printf("missing or unknown storage_type. using 'filesystem'")
		fallthrough
	case "filesystem":
		store, err = file.New(cfg.StorageConfig)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
	}

	// Encoder setup
	enc, err := encoder.New(cfg.StreamConfig)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	streamsDir, ok := cfg.StreamConfig["media_dir"]
	if !ok {
		log.Printf("must set stream_config.media_dir")
		os.Exit(1)
	}
	const streamRoute = "/stream"

	h := handlers{
		Config:      cfg,
		Store:       store,
		Encoder:     enc,
		Relay:       relay,
		StreamRoute: streamRoute,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(cors.AllowAll().Handler)
	r.Use(metricsMiddleware)

	r.Get("/{sum}/{name}", h.handleGetMedia)
	r.Get("/{sum}", h.handleGetMedia)
	r.Get("/upload/quote", h.handleGetQuote)
	r.Post("/upload", h.handleUploadMedia)
	r.Method(http.MethodGet, "/metrics", promhttp.Handler())

	fileServer(r, streamRoute, http.Dir(streamsDir))

	port := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("listening on %v\n", port)

	http.ListenAndServe(port, r)
}
