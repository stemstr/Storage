package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/nbd-wtf/go-nostr"
)

type handlers struct {
	Config      Config
	Store       storageProvider
	Encoder     encoderProvider
	Relay       nostrProvider
	StreamRoute string
}

// handleDownloadMedia fetches stored media
func (h *handlers) handleDownloadMedia(w http.ResponseWriter, r *http.Request) {
	var (
		ctx = r.Context()
		sum = chi.URLParam(r, "sum")
	)

	f, err := h.Store.Get(ctx, sum)
	if err != nil {
		log.Printf("err: store.Get: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fileBytes, err := ioutil.ReadAll(f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	downloadCounter.Inc()
	w.Header().Set("Content-Type", http.DetectContentType(fileBytes))
	w.Header().Set("Content-Length", strconv.Itoa(len(fileBytes)))
	w.Write(fileBytes)
}

// handleGetQuote returns an upload quote
func (h *handlers) handleGetQuote(w http.ResponseWriter, r *http.Request) {
	var (
		pubkey   = r.URL.Query().Get("pk")
		fileSize = r.URL.Query().Get("size")
		sum      = r.URL.Query().Get("sig")
	)

	if pubkey == "" {
		http.Error(w, "pk param required", http.StatusBadRequest)
		return
	}
	if fileSize == "" {
		http.Error(w, "size param required", http.StatusBadRequest)
		return
	}
	if sum == "" {
		http.Error(w, "sig param required", http.StatusBadRequest)
		return
	}

	// TODO: Calculate price
	// TODO: Create invoice

	// We bake the final stream and download urls into the event so we must
	// calculate them now, before the file is actually uploaded.
	streamPath, _ := url.JoinPath(h.Config.MediaPath, h.StreamRoute, sum+".m3u8")
	downloadPath, _ := url.JoinPath(h.Config.MediaPath, sum)

	event := newAudioEvent(pubkey, streamPath, downloadPath)

	data, err := json.Marshal(map[string]any{
		"invoice": "",
		"event":   event,
	})

	if err != nil {
		log.Printf("failed to marshal quote: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// handleUploadMedia stores the provided media
func (h *handlers) handleUploadMedia(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.Config.MaxUploadSizeMB*1024*1024)

	// TODO: Get the signed event

	payload, err := h.getMedia(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var (
		contentType = http.DetectContentType(payload.Data)
		accepted    = false
	)
	if len(h.Config.AcceptedMimetypes) == 0 {
		// No explicit accepted mimetypes, allow all.
		accepted = true
	} else {
		for _, mime := range h.Config.AcceptedMimetypes {
			if strings.EqualFold(contentType, mime) {
				accepted = true
				break
			}
		}
	}
	if !accepted {
		msg := fmt.Sprintf("unaccepted content mimetype %q", contentType)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	var (
		ctx = r.Context()
		sum = fmt.Sprintf("%x", sha256.Sum256(payload.Data))
	)

	if err := h.Store.Save(ctx, bytes.NewReader(payload.Data), sum); err != nil {
		log.Printf("err: store.Save: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: Move encoding into async worker pool.
	filePath := filepath.Join("./files", sum, "data")
	_, err = h.Encoder.EncodeMP3(ctx, filePath, contentType, sum)
	if err != nil {
		log.Printf("err: encodeMP3: %v", err)
		http.Error(w, "please try again later", http.StatusInternalServerError)
		return
	}

	// TODO: Only do this after async encoder pool has completed
	_ = h.Relay.Publish(ctx, payload.Event)

	data, err := json.Marshal(map[string]any{
		"data":    map[string]any{},
		"success": true,
		"status":  200,
	})
	if err != nil {
		log.Printf("failed to marshal upload response: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	uploadCounter.Inc()

	w.WriteHeader(http.StatusCreated)
	w.Write(data)
}

type uploadRequest struct {
	Pubkey string
	Event  nostr.Event
	Data   []byte
}

func (h *handlers) getMedia(r *http.Request) (*uploadRequest, error) {
	err := r.ParseMultipartForm(h.Config.MaxUploadSizeMB * 1024 * 1024)
	if err != nil {
		return nil, err
	}

	pk := r.Form.Get("pk")
	if pk == "" {
		return nil, fmt.Errorf("must provide pk field")
	}

	eventStr := r.Form.Get("event")
	if eventStr == "" {
		return nil, fmt.Errorf("must provide event field")
	}
	event, err := parseEncodedEvent(eventStr)
	if err != nil {
		return nil, fmt.Errorf("must provide valid event field")
	}

	f, _, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fileBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return &uploadRequest{
		Pubkey: pk,
		Event:  *event,
		Data:   fileBytes,
	}, nil
}

func fileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
