package main

import (
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
	"github.com/nbd-wtf/go-nostr"

	"github.com/stemstr/storage/internal/service"
)

type handlers struct {
	config Config
	svc    *service.Service
	relay  nostrProvider
}

// handleDownloadMedia fetches stored media
func (h *handlers) handleDownloadMedia(w http.ResponseWriter, r *http.Request) {
	var (
		ctx = r.Context()
		sum = chi.URLParam(r, "sum")
	)

	resp, err := h.svc.GetSample(ctx, sum)
	if err != nil {
		log.Printf("err: svc.GetSample: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	downloadCounter.Inc()
	w.Header().Set("Content-Disposition", "attachment; filename="+resp.Filename)
	w.Header().Set("Content-Length", strconv.Itoa(len(resp.Data)))
	w.Header().Set("Content-Type", resp.ContentType)
	w.Write(resp.Data)
}

// handleGetMetadata fetches stored media metadata
func (h *handlers) handleGetMetadata(w http.ResponseWriter, r *http.Request) {
	var (
		ctx = r.Context()
		sum = chi.URLParam(r, "sum")
	)

	resp, err := h.svc.GetSample(ctx, sum)
	if err != nil {
		log.Printf("err: svc.GetSample: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(map[string]any{
		"waveform": resp.Media.Waveform,
	})

	if err != nil {
		log.Printf("failed to marshal resp: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

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
	downloadPath, _ := url.JoinPath(h.config.DownloadBase, resp.MediaID)

	data, err := json.Marshal(map[string]any{
		"stream_url":   streamPath,
		"download_url": downloadPath,
		"waveform":     resp.Waveform,
	})

	if err != nil {
		log.Printf("failed to marshal resp: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

	f, _, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fileBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	mimeType := detectContentType(fileBytes, &fileName)

	return &service.NewSampleRequest{
		Data:     fileBytes,
		Mimetype: mimeType,
		Sum:      sum,
		Pubkey:   pk,
	}, nil
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
	if !validPubkey(req.PubKey) {
		http.Error(w, "invalid pubkey", http.StatusBadRequest)
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

	h.relay.Publish(ctx, event)
	w.WriteHeader(http.StatusOK)
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

func validPubkey(pk string) bool {
	return len(pk) == 64
}
