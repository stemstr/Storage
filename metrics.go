package main

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	uploadCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "uploads",
		Help: "The total number of uploads",
	})
	downloadCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "downloads",
		Help: "The total number of files fetched",
	})
	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_response_duration_seconds",
		Help: "Latency of requests in second.",
	}, []string{"path"})
)

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timer := prometheus.NewTimer(httpDuration.WithLabelValues(r.URL.Path))

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		timer.ObserveDuration()
	})
}
