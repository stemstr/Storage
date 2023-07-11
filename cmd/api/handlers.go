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

	"github.com/go-chi/chi/v5"

	"github.com/stemstr/storage/internal/mimes"
	"github.com/stemstr/storage/internal/service"
)

type handlers struct {
	config Config
	svc    *service.Service
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

	sample, err := h.svc.GetSample(ctx, filename)
	if err != nil {
		log.Printf("err: svc.GetSample: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+sample.Filename)
	w.Header().Set("Content-Length", strconv.Itoa(len(sample.Data)))
	w.Header().Set("Content-Type", sample.ContentType)
	w.Header().Set("X-Download-Filename", sample.Filename)
	w.Write(sample.Data)
}

// handleGetStream redirects requests for stream files to the new CDN.
// Some early notes have a stream_url pointed at the api.
func (h *handlers) handleGetStream(w http.ResponseWriter, r *http.Request) {
	var filename = chi.URLParam(r, "filename")

	cdnURL, _ := url.JoinPath(h.config.CDNHost, "stream", filename)
	http.Redirect(w, r, cdnURL, http.StatusTemporaryRedirect)
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

	sample, err := h.svc.NewSample(ctx, req)
	if err != nil {
		log.Printf("svc.NewSample: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, _ := json.Marshal(map[string]any{
		"stream_url":   sample.StreamURL(h.config.CDNHost),
		"download_url": sample.DownloadURL(h.config.APIHost),
		"waveform":     sample.Waveform,
	})
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
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
