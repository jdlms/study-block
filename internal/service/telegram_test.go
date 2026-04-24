package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTelegramTestService(rt roundTripFunc) *TelegramService {
	return &TelegramService{
		token:      "test-token",
		httpClient: &http.Client{Transport: rt},
	}
}

func TestGetUpdatesRejectsHTTPErrorStatus(t *testing.T) {
	svc := newTelegramTestService(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Status:     "401 Unauthorized",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"ok":false}`)),
			Request:    req,
		}, nil
	})

	_, err := svc.getUpdates(context.Background(), 0)
	if err == nil || !strings.Contains(err.Error(), "401 Unauthorized") {
		t.Fatalf("err = %v, want status error", err)
	}
}

func TestGetUpdatesRejectsTelegramErrorBody(t *testing.T) {
	svc := newTelegramTestService(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"ok":false,"error_code":429,"description":"Too Many Requests"}`)),
			Request:    req,
		}, nil
	})

	_, err := svc.getUpdates(context.Background(), 0)
	if err == nil || !strings.Contains(err.Error(), "Too Many Requests") {
		t.Fatalf("err = %v, want telegram error", err)
	}
}

func TestGetUpdatesReturnsParsedUpdates(t *testing.T) {
	svc := newTelegramTestService(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
				"ok": true,
				"result": [{"update_id": 7, "message": {"message_id": 8, "date": 9, "chat": {"id": 10}, "text": "25 math"}}]
			}`)),
			Request: req,
		}, nil
	})

	updates, err := svc.getUpdates(context.Background(), 0)
	if err != nil {
		t.Fatalf("getUpdates() error = %v", err)
	}
	if len(updates) != 1 || updates[0].UpdateID != 7 || updates[0].Message.Text != "25 math" {
		t.Fatalf("updates = %#v, want parsed update", updates)
	}
}
