package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/stemstr/storage/internal/mimes"
	"github.com/stemstr/storage/internal/service"
	"github.com/stemstr/storage/internal/subscription"
)

type handlers struct {
	config Config
	svc    *service.Service
	subs   *subscription.SubscriptionService
	blastr blastrIface
}

type blastrIface interface {
	SendText(context.Context, string) error
}

// handleDownloadMedia fetches stored media
func (h *handlers) handleDownloadMedia(w http.ResponseWriter, r *http.Request) {
	var (
		ctx      = r.Context()
		filename = chi.URLParam(r, "filename")
	)

	// Some early notes did not have a file extension on the download URL.
	if !strings.HasSuffix(filename, ".wav") {
		filename += ".wav"
	}

	resp, err := h.svc.GetSample(ctx, filename)
	if err != nil {
		log.Printf("err: svc.GetSample: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	downloadCounter.Inc()
	w.Header().Set("Content-Disposition", "attachment; filename="+resp.Filename)
	w.Header().Set("Content-Length", strconv.Itoa(len(resp.Data)))
	w.Header().Set("Content-Type", resp.ContentType)
	w.Header().Set("X-Download-Filename", resp.Filename)
	w.Write(resp.Data)
}

// handleGetStream redirects requests for stream files to the new CDN.
// Some early notes have a stream_url pointed at the api.
func (h *handlers) handleGetStream(w http.ResponseWriter, r *http.Request) {
	var filename = chi.URLParam(r, "filename")

	cdnURL, _ := url.JoinPath("https://cdn.stemstr.app/stream", filename)
	http.Redirect(w, r, cdnURL, http.StatusTemporaryRedirect)
}

// handleGetSubscriptionOptions returns subscription options
func (h *handlers) handleGetSubscriptionOptions(w http.ResponseWriter, r *http.Request) {
	jsonb, _ := json.Marshal(h.config.SubscriptionOptions)

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonb)
	return
}

// handleGetSubscription fetches the active subscription for a pubkey
func (h *handlers) handleGetSubscription(w http.ResponseWriter, r *http.Request) {
	var (
		ctx    = r.Context()
		pubkey = chi.URLParam(r, "pubkey")
	)

	sub, err := h.subs.GetActiveSubscription(ctx, pubkey)
	if err != nil {
		switch {
		case errors.Is(err, subscription.ErrSubscriptionNotFound):
			log.Printf("sub not found: pk=%v\n", pubkey)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		case errors.Is(err, subscription.ErrSubscriptionExpired):
			log.Printf("sub expired: pk=%v\n", pubkey)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		default:
			log.Printf("err: subs.GetSubscriptionStatus: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	jsonb, _ := json.Marshal(map[string]any{
		"days":       sub.Days,
		"created_at": sub.CreatedAt.Unix(),
		"expires_at": sub.ExpiresAt.Unix(),
	})

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonb)
}

// handleCreateSubscription creates a new subscription. If pubkey already
// has an active subscription, it will be returned.
func (h *handlers) handleCreateSubscription(w http.ResponseWriter, r *http.Request) {
	var (
		ctx     = r.Context()
		pubkey  = chi.URLParam(r, "pubkey")
		daysStr = r.URL.Query().Get("days")
	)

	if daysStr == "" {
		http.Error(w, "must provide days query param", http.StatusBadRequest)
		return
	}
	days, err := strconv.Atoi(daysStr)
	if err != nil {
		http.Error(w, "days must be a valid subscription days", http.StatusBadRequest)
		return
	}

	subOptions := map[int]int{}
	for _, opt := range h.config.SubscriptionOptions {
		subOptions[opt.Days] = opt.Sats
	}

	sats, ok := subOptions[days]
	if !ok {
		http.Error(w, "invalid subscription days", http.StatusBadRequest)
		return
	}
	now := time.Now()
	expiry := now.Add(time.Hour * 24 * time.Duration(days))

	existingSub, err := h.subs.GetActiveSubscription(ctx, pubkey)
	if err != nil {
		switch {
		case errors.Is(err, subscription.ErrSubscriptionNotFound):
			// noop
			log.Printf("sub not found: pk=%v\n", pubkey)
		case errors.Is(err, subscription.ErrSubscriptionExpired):
			// noop
			log.Printf("sub expired: pk=%v\n", pubkey)
		default:
			log.Printf("err: subs.GetSubscriptionStatus: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if existingSub != nil {
		log.Printf("createSubscription: already exists! %#v", *existingSub)
		http.Error(w, "active subscription", http.StatusConflict)
		return
	}

	sub, err := h.subs.CreateSubscription(ctx, subscription.Subscription{
		Pubkey:    pubkey,
		Days:      days,
		Sats:      sats,
		CreatedAt: now,
		ExpiresAt: expiry,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonb, _ := json.Marshal(map[string]any{
		"lightning_invoice": sub.LightningInvoice,
		"days":              sub.Days,
		"created_at":        sub.CreatedAt.Unix(),
		"expires_at":        sub.ExpiresAt.Unix(),
	})

	w.WriteHeader(http.StatusPaymentRequired)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonb)
	return
}

func (h *handlers) handleCallbackZBDCharge(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Amount      string `json:"amount"`      // "1000000"
		ConfirmedAt string `json:"confirmedAt"` // "2023-07-31T21:14:44.000Z"
		CreatedAt   string `json:"createdAt"`   // "2023-07-31T21:14:33.184Z"
		Description string `json:"description"` // "Stemstr 1 day subscription"
		ExpiresAt   string `json:"expiresAt"`   // "2023-07-31T21:19:33.163Z"
		ID          string `json:"id"`          // "077c6d70-421f-4a5c-9baa-85c80ec11ace"
		InternalID  string `json:"internalId"`  // "0"
		Invoice     struct {
			Request string `json:"request"` // "lnbc10u1pjvsfpepp597...",
			URI     string `json:"uri"`     // "lightning:lnbc10u1pj..."
		} `json:"invoice"`
		Status string `json:"status"` // "completed"
		Unit   string `json:"unit"`   // "msats"
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "expected JSON payload", http.StatusBadRequest)
		return
	}

	if data.Status == "completed" {
		ctx := r.Context()
		err := h.subs.UpdateInvoiceStatus(ctx, data.ID, subscription.StatusPaid)
		if err != nil {
			log.Printf("error: updateInvoiceStatus: invoice_id=%v err=%v", data.ID, err.Error())
			http.Error(w, "unable to update invoice status", http.StatusInternalServerError)
			return
		}

		go func() {
			if h.blastr != nil {
				h.blastr.SendText(context.Background(), "new subscription")
			}
		}()
	}

	w.WriteHeader(http.StatusOK)
}

// handleUpload handles user media uploads
func (h *handlers) handleUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	r.Body = http.MaxBytesReader(w, r.Body, h.config.MaxUploadSizeMB*1024*1024)

	req, err := h.parseUploadRequest(r)
	if err != nil {
		if errors.Is(err, ErrLogin) {
			http.Error(w, err.Error(), http.StatusUnauthorized)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	if _, err := h.subs.GetActiveSubscription(ctx, req.Pubkey); err != nil {
		log.Printf("upload blocked: subscription not found for %q, err: %v", req.Pubkey, err)
		http.Error(w, "Subscription required", http.StatusPaymentRequired)
		return
	}

	sum := fmt.Sprintf("%x", sha256.Sum256(req.Data))
	if req.Sum != sum {
		http.Error(w, "sum does not match content", http.StatusBadRequest)
		return
	}

	if !validPubkey(req.Pubkey) {
		http.Error(w, "invalid pubkey", http.StatusBadRequest)
		return
	}

	accepted := false
	if len(h.config.AcceptedMimetypes) == 0 {
		// No explicit accepted mimetypes, allow all.
		accepted = true
	} else {
		for _, mime := range h.config.AcceptedMimetypes {
			if strings.EqualFold(req.Mimetype, mime) {
				accepted = true
				break
			}
		}
	}
	if !accepted {
		log.Printf("unaccepted mimetype %q\n", req.Mimetype)
		http.Error(w, "unaccepted content type", http.StatusBadRequest)
		return
	}

	resp, err := h.svc.NewSample(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	streamPath, _ := url.JoinPath(h.config.StreamBase, resp.MediaID+".m3u8")
	downloadPath, _ := url.JoinPath(h.config.DownloadBase, resp.MediaID+".wav")

	data, err := json.Marshal(map[string]any{
		"stream_url":    streamPath,
		"download_url":  downloadPath,
		"download_hash": resp.DownloadHash,
		"waveform":      resp.Waveform,
	})

	if err != nil {
		log.Printf("failed to marshal resp: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	uploadCounter.Inc()
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

var (
	ErrLogin = fmt.Errorf("login required")
)

func (h *handlers) parseUploadRequest(r *http.Request) (*service.NewSampleRequest, error) {
	err := r.ParseMultipartForm(h.config.MaxUploadSizeMB * 1024 * 1024)
	if err != nil {
		return nil, err
	}

	// Required form fields
	// pk, sum, filename, file

	pk := r.Form.Get("pk")
	if pk == "" {
		return nil, ErrLogin
	}

	sum := r.Form.Get("sum")
	if sum == "" {
		return nil, fmt.Errorf("must provide sum field")
	}

	fileName := r.Form.Get("filename")
	if fileName == "" {
		return nil, fmt.Errorf("must provide filename field")
	}

	mimeType := mimes.FromFilename(fileName)
	if mimeType == "" {
		return nil, fmt.Errorf("unaccepted audio file: %q", fileName)
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

	return &service.NewSampleRequest{
		Data:     fileBytes,
		Mimetype: mimeType,
		Sum:      sum,
		Pubkey:   pk,
	}, nil
}

func pubkeyIsAllowed(pubkeys []string, pubkey string) bool {
	// If no whitelist of pubkeys are provided, it's allowed
	if len(pubkeys) == 0 {
		return true
	}

	allowed := false
	for _, allowedPubkey := range pubkeys {
		if strings.EqualFold(allowedPubkey, pubkey) {
			allowed = true
			break
		}
	}

	return allowed
}

func (h *handlers) handleDebugStream(w http.ResponseWriter, r *http.Request) {
	const html = `<html>
	<head>
		<title>Debug stream</title>
    <script src="https://hlsjs-dev.video-dev.org/dist/hls.js"></script>
	</head>
  <body>
    <center>
      <h1>Debug stream</h1>
      <div>
        <input id="url" placeholder="stream url">
        <button onClick="loadStream()">load</button>
      </div>

			<video controls id="video" height="600"></video>
    </center>

    <script>
      const doIt = (url) => {
        var video = document.getElementById('video');
        if (Hls.isSupported()) {
          var hls = new Hls({
            debug: true,
          });
          hls.loadSource(url);
          hls.attachMedia(video);
          hls.on(Hls.Events.MEDIA_ATTACHED, function () {
            video.muted = true;
            video.play();
          });
        }
      }
      const loadStream = () => {
        const url = document.getElementById("url").value;
        doIt(url)
      }
    </script>
	</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func validPubkey(pk string) bool {
	return len(pk) == 64
}
