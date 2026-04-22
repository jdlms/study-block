package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"study-blocks/internal/service"
	"study-blocks/internal/store"
)

type APIHandler struct {
	ingest                *service.IngestService
	enableLocalTestRoutes bool
}

func NewAPIHandler(ingest *service.IngestService, enableLocalTestRoutes bool) *APIHandler {
	return &APIHandler{ingest: ingest, enableLocalTestRoutes: enableLocalTestRoutes}
}

func (h *APIHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", h.health)
	mux.HandleFunc("/api/subjects", h.subjects)
	mux.HandleFunc("/api/entries", h.entries)
	mux.HandleFunc("/api/entries/", h.entryByID)
	mux.HandleFunc("/api/testing/clear", h.clear)
}

func (h *APIHandler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
func (h *APIHandler) subjects(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.ingest.Subjects())
}

func (h *APIHandler) entries(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")
		if !validDate(from) || !validDate(to) {
			http.Error(w, "from and to must be YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		entries, err := h.ingest.ListEntries(from, to)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, entries)
	case http.MethodPost:
		var req struct {
			Date    string `json:"date"`
			Subject string `json:"subject"`
			Minutes int    `json:"minutes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if !validDate(req.Date) {
			http.Error(w, "date must be YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		entry, err := h.ingest.CreateEntry(req.Date, req.Subject, req.Minutes, time.Now())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusCreated, entry)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *APIHandler) entryByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/entries/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if err := h.ingest.DeleteEntry(id); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrNotFound) {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *APIHandler) clear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !h.enableLocalTestRoutes || !isLocalRequest(r) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := h.ingest.ClearEntries(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func validDate(v string) bool {
	_, err := time.Parse(time.DateOnly, v)
	return err == nil
}

func isLocalRequest(r *http.Request) bool {
	host := r.RemoteAddr
	return strings.HasPrefix(host, "127.0.0.1:") || strings.HasPrefix(host, "[::1]:") || strings.HasPrefix(host, "localhost:")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
