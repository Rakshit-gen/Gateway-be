package handlers

import (
	"encoding/json"
	"fmt"
	"gateway/internal/analytics"
	"net/http"
	"time"
)

type AnalyticsHandler struct {
	analytics *analytics.Analytics
}

func NewAnalyticsHandler(analytics *analytics.Analytics) *AnalyticsHandler {
	return &AnalyticsHandler{analytics: analytics}
}

func (h *AnalyticsHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	var startTime, endTime time.Time
	var err error

	if startStr != "" {
		startTime, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			http.Error(w, `{"error":"invalid start time format"}`, http.StatusBadRequest)
			return
		}
	} else {
		startTime = time.Now().Add(-24 * time.Hour)
	}

	if endStr != "" {
		endTime, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			http.Error(w, `{"error":"invalid end time format"}`, http.StatusBadRequest)
			return
		}
	} else {
		endTime = time.Now()
	}

	metrics, err := h.analytics.GetMetrics(r.Context(), startTime, endTime)
	if err != nil {
		http.Error(w, `{"error":"failed to get metrics"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (h *AnalyticsHandler) StreamMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
		return
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			metrics, err := h.analytics.GetRealtimeMetrics(r.Context())
			if err != nil {
				continue
			}

			data, err := json.Marshal(metrics)
			if err != nil {
				continue
			}

			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}
