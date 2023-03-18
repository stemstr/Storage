package main

import (
	"bytes"
	"context"
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
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/nbd-wtf/go-nostr"

	db "github.com/stemstr/storage/internal/db/sqlite"
)

type handlers struct {
	Config  Config
	Store   storageProvider
	Encoder encoderProvider
	Relay   nostrProvider
	DB      db.DB
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

// handlePostEvent proxies an event to the relay.
func (h *handlers) handlePostEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		ID        string     `json:"id"`
		PubKey    string     `json:"pubkey"`
		CreatedAt int        `json:"created_at"`
		Kind      int        `json:"kind"`
		Tags      [][]string `json:"tags"`
		Content   string     `json:"content"`
		Sig       string     `json:"sig"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "unable to parse request", http.StatusBadRequest)
		return
	}

	createdAt := time.Unix(int64(req.CreatedAt), 0)

	event := nostr.Event{
		ID:        req.ID,
		PubKey:    req.PubKey,
		CreatedAt: createdAt,
		Kind:      req.Kind,
		Tags:      parseTags(req.Tags),
		Content:   req.Content,
		Sig:       req.Sig,
	}

	valid, err := event.CheckSignature()
	if err != nil || !valid {
		http.Error(w, "invalid event", http.StatusBadRequest)
		return
	}

	_ = h.Relay.Publish(ctx, event)
	w.WriteHeader(http.StatusOK)
}

type getQuoteRequest struct {
	Pubkey   string `json:"pk"`
	Filesize int64  `json:"size"`
	Sum      string `json:"sum"`
	Desc     string `json:"desc"`
}

// handleGetQuote returns an upload quote
func (h *handlers) handleGetQuote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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
	sats := int64(0)
	// TODO: Create LN invoice
	lnProvider := "fixme_some_ln_provider"
	paymentHash := "fixme_rhash"

	// Create invoice
	invoice, err := h.DB.CreateInvoice(ctx, db.CreateInvoiceRequest{
		PaymentHash: paymentHash,
		Paid:        false,
		Sats:        sats,
		Provider:    lnProvider,
		CreatedBy:   req.Pubkey,
	})
	if err != nil {
		log.Printf("error: create invoice: %v", err)
		http.Error(w, "failed to create invoice", http.StatusInternalServerError)
		return
	}

	// Create Media record
	media, err := h.DB.CreateMedia(ctx, db.CreateMediaRequest{
		InvoiceID: invoice.ID,
		Size:      req.Filesize,
		Sum:       req.Sum,
		CreatedBy: req.Pubkey,
	})
	if err != nil {
		http.Error(w, "failed to create quote", http.StatusInternalServerError)
		return
	}

	// We bake the final stream and download urls into the event so we must
	// calculate them now, before the file is actually uploaded.
	streamPath, _ := url.JoinPath(h.Config.StreamBase, media.Sum+".m3u8")
	downloadPath, _ := url.JoinPath(h.Config.DownloadBase, media.Sum)

	data, err := json.Marshal(map[string]any{
		"invoice": "",
		// TODO: rename to invoice_id?
		"quote_id":     invoice.ID,
		"stream_url":   streamPath,
		"download_url": downloadPath,
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
	ctx := r.Context()
	r.Body = http.MaxBytesReader(w, r.Body, h.Config.MaxUploadSizeMB*1024*1024)

	upload, err := h.parseUploadRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sum := upload.ShaSum()

	invoice, media, err := h.validateUpload(ctx, upload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Printf("got invoice: %#v media: %#v\n", *invoice, *media)

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
	Size        int64
	Sum         string
	QuoteID     string
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
	size, err := strconv.ParseInt(sizeStr, 10, 64)
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
	quoteID := r.Form.Get("quoteId")
	if quoteID == "" {
		return nil, fmt.Errorf("must provide quoteId field")
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
		QuoteID:     quoteID,
		ContentType: contentType,
		FileName:    fileName,
		Data:        fileBytes,
	}, nil
}

func (h *handlers) validateUpload(ctx context.Context, payload *uploadRequest) (*db.Invoice, *db.Media, error) {
	// TODO: tx
	invoice, err := h.DB.GetInvoice(ctx, payload.QuoteID)
	if err != nil {
		log.Printf("error: get invoice: %v", err)
		return nil, nil, err
	}

	media, err := h.DB.GetMediaByInvoice(ctx, invoice.ID)
	if err != nil {
		log.Printf("error: get media: %v", err)
		return nil, nil, err
	}

	if payload.Pubkey != invoice.CreatedBy {
		return nil, nil, fmt.Errorf("err: pubkey invalid")
	}
	if payload.Sum != media.Sum {
		return nil, nil, fmt.Errorf("err: sum invalid")
	}
	if payload.Size != media.Size {
		return nil, nil, fmt.Errorf("err: size invalid")
	}

	valid, err := payload.Event.CheckSignature()
	if err != nil || !valid {
		return nil, nil, fmt.Errorf("err: event signature invalid")
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
		return nil, nil, fmt.Errorf("unaccepted content mimetype %q", contentType)
	}

	return invoice, media, nil
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
