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
	Config  Config
	Store   storageProvider
	Encoder encoderProvider
	Relay   nostrProvider
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
	w.Header().Set("Content-Type", detectContentType(fileBytes, nil))
	w.Header().Set("Content-Length", strconv.Itoa(len(fileBytes)))
	w.Write(fileBytes)
}

type getQuoteRequest struct {
	Pubkey   string   `json:"pk"`
	Filesize int      `json:"size"`
	Sum      string   `json:"sum"`
	Desc     string   `json:"desc"`
	Tags     []string `json:"tags"`
}

// handleGetQuote returns an upload quote
func (h *handlers) handleGetQuote(w http.ResponseWriter, r *http.Request) {
	var req getQuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "unable to parse request", http.StatusBadRequest)
		return
	}

	if req.Pubkey == "" {
		http.Error(w, "pk field required", http.StatusBadRequest)
		return
	}
	if req.Filesize == 0 {
		http.Error(w, "size field required", http.StatusBadRequest)
		return
	}
	if req.Sum == "" {
		http.Error(w, "sum field required", http.StatusBadRequest)
		return
	}

	// TODO: Calculate price
	// TODO: Create invoice

	// We bake the final stream and download urls into the event so we must
	// calculate them now, before the file is actually uploaded.
	streamPath, _ := url.JoinPath(h.Config.StreamBase, req.Sum+".m3u8")
	downloadPath, _ := url.JoinPath(h.Config.DownloadBase, req.Sum)

	event := newAudioEvent(req.Pubkey, req.Desc, req.Tags, streamPath, downloadPath)

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
	upload, err := h.parseUploadRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.validateUpload(upload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var (
		ctx = r.Context()
		sum = upload.ShaSum()
	)

	err = h.Store.Save(ctx, bytes.NewReader(upload.Data), sum)
	if err != nil {
		log.Printf("err: store.Save: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: Move encoding into async worker pool.
	var (
		filePath    = filepath.Join("./files", sum, "data")
		contentType = upload.ContentType
	)
	_, err = h.Encoder.EncodeMP3(ctx, filePath, contentType, sum)
	if err != nil {
		log.Printf("err: encodeMP3: %v", err)
		http.Error(w, "please try again later", http.StatusInternalServerError)
		return
	}

	// TODO: Only do this after async encoder pool has completed
	_ = h.Relay.Publish(ctx, upload.Event)

	uploadCounter.Inc()

	w.WriteHeader(http.StatusCreated)
}

type uploadRequest struct {
	Pubkey      string
	Event       nostr.Event
	Size        int
	Sum         string
	ContentType string
	FileName    string
	Data        []byte
}

func (r *uploadRequest) ShaSum() string {
	return fmt.Sprintf("%x", sha256.Sum256(r.Data))
}

func (h *handlers) parseUploadRequest(r *http.Request) (*uploadRequest, error) {
	err := r.ParseMultipartForm(h.Config.MaxUploadSizeMB * 1024 * 1024)
	if err != nil {
		return nil, err
	}

	pk := r.Form.Get("pk")
	if pk == "" {
		return nil, fmt.Errorf("must provide pk field")
	}
	sizeStr := r.Form.Get("size")
	if sizeStr == "" {
		return nil, fmt.Errorf("must provide size field")
	}
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		return nil, fmt.Errorf("size field must be an int")
	}
	sum := r.Form.Get("sum")
	if sum == "" {
		return nil, fmt.Errorf("must provide sum field")
	}
	fileName := r.Form.Get("fileName")
	if fileName == "" {
		return nil, fmt.Errorf("must provide fileName field")
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

	contentType := detectContentType(fileBytes, &fileName)

	return &uploadRequest{
		Pubkey:      pk,
		Event:       *event,
		Size:        size,
		Sum:         sum,
		ContentType: contentType,
		FileName:    fileName,
		Data:        fileBytes,
	}, nil
}

func (h *handlers) validateUpload(payload *uploadRequest) error {
	valid, err := payload.Event.CheckSignature()
	if err != nil || !valid {
		return fmt.Errorf("err: event signature invalid")
	}

	var (
		contentType = detectContentType(payload.Data, &payload.FileName)
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
		return fmt.Errorf("unaccepted content mimetype %q", contentType)
	}

	return nil
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

func detectContentType(data []byte, fileName *string) string {
	if fileName != nil {
		switch {
		case strings.HasSuffix(*fileName, ".m4a"):
			// http.DetectContentType will return "vidio/mp4" for MPEG-4 audio
			return "audio/mp4"
		case strings.HasSuffix(*fileName, ".mp3"):
			// http.DetectContentType will return "application/octet-stream" for some MP3's
			return "audio/mp3"
		}
	}

	return http.DetectContentType(data)
}
