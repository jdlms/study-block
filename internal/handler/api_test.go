package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"study-blocks/internal/model"
	"study-blocks/internal/service"
	"study-blocks/internal/store"
)

func newTestAPIHandler(t *testing.T) *APIHandler {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ingest := service.NewIngestService(db, []model.Subject{{Name: "math", Color: "#111"}, {Name: "physics", Color: "#222"}})
	entry1, err := ingest.CreateEntry("2026-04-21", "math", 45, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	_, err = ingest.CreateEntry("2026-04-22", "physics", 20, time.Unix(200, 0))
	if err != nil {
		t.Fatal(err)
	}
	_ = entry1
	return NewAPIHandler(ingest, true)
}

func TestEntriesGET(t *testing.T) {
	h := newTestAPIHandler(t)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/entries?from=2026-04-22&to=2026-04-22", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	var entries []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
}

func TestClearLocalTestingRoute(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ingest := service.NewIngestService(db, []model.Subject{{Name: "math", Color: "#111"}})
	if _, err := ingest.CreateEntry("2026-04-22", "math", 30, time.Unix(100, 0)); err != nil {
		t.Fatal(err)
	}
	h := NewAPIHandler(ingest, true)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/testing/clear", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", res.Code, res.Body.String())
	}

	entries, err := ingest.ListEntries("2026-04-01", "2026-04-30")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("len(entries) = %d, want 0", len(entries))
	}
}

func TestEntriesPOST(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ingest := service.NewIngestService(db, []model.Subject{{Name: "math", Color: "#111"}})
	h := NewAPIHandler(ingest, true)
	mux := http.NewServeMux()
	h.Register(mux)

	body := strings.NewReader(`{"date":"2026-04-22","subject":"math","minutes":30}`)
	req := httptest.NewRequest(http.MethodPost, "/api/entries", body)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 body=%s", res.Code, res.Body.String())
	}
}
