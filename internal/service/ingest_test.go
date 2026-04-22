package service

import (
	"path/filepath"
	"testing"
	"time"

	"study-blocks/internal/model"
	"study-blocks/internal/store"
)

func newTestIngestService(t *testing.T) *IngestService {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewIngestService(db, []model.Subject{{Name: "math", Color: "#111"}, {Name: "physics", Color: "#222"}})
}

func TestParseTelegramMessage(t *testing.T) {
	svc := newTestIngestService(t)
	ts := time.Unix(1710000000, 0)

	entry, err := svc.ParseTelegramMessage("25 Math", ts)
	if err != nil {
		t.Fatalf("ParseTelegramMessage() error = %v", err)
	}
	if entry.Subject != "math" {
		t.Fatalf("subject = %q, want math", entry.Subject)
	}
	if entry.Minutes != 25 {
		t.Fatalf("minutes = %d, want 25", entry.Minutes)
	}
	if entry.Date != ts.Format(time.DateOnly) {
		t.Fatalf("date = %q, want %q", entry.Date, ts.Format(time.DateOnly))
	}
}

func TestParseTelegramMessageRejectsInvalidInput(t *testing.T) {
	svc := newTestIngestService(t)
	ts := time.Now()

	cases := []string{"", "math", "0 math", "x math", "25 history", "25 math extra"}
	for _, tc := range cases {
		if _, err := svc.ParseTelegramMessage(tc, ts); err == nil {
			t.Fatalf("ParseTelegramMessage(%q) expected error", tc)
		}
	}
}

func TestUndoRemovesLatestEntryForDate(t *testing.T) {
	svc := newTestIngestService(t)
	day := "2026-04-22"

	_, _ = svc.CreateEntry(day, "math", 15, time.Unix(100, 0))
	latest, _ := svc.CreateEntry(day, "physics", 25, time.Unix(200, 0))

	undone, err := svc.Undo(day)
	if err != nil {
		t.Fatal(err)
	}
	if undone == nil || undone.ID != latest.ID {
		t.Fatalf("Undo() = %#v, want latest entry %#v", undone, latest)
	}
}
