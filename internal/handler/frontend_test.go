package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFrontendRouteFallsBackToIndex(t *testing.T) {
	mux := http.NewServeMux()
	if err := RegisterFrontend(mux); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/history/today", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	if !strings.Contains(res.Body.String(), `<div id="app"></div>`) {
		t.Fatalf("body = %q, want index.html", res.Body.String())
	}
}

func TestFrontendMissingAssetReturnsNotFound(t *testing.T) {
	mux := http.NewServeMux()
	if err := RegisterFrontend(mux); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/assets/missing.js", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 body=%s", res.Code, res.Body.String())
	}
}
