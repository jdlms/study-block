package store

import (
	"errors"
	"path/filepath"
	"testing"

	"study-blocks/internal/model"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestListEntriesFiltersAndSorts(t *testing.T) {
	s := newTestStore(t)
	entries := []model.Entry{
		{ID: "3", Timestamp: 300, Date: "2026-04-23", Subject: "math", Minutes: 15},
		{ID: "1", Timestamp: 100, Date: "2026-04-22", Subject: "math", Minutes: 25},
		{ID: "2", Timestamp: 200, Date: "2026-04-22", Subject: "physics", Minutes: 40},
	}
	for _, entry := range entries {
		if err := s.CreateEntry(entry); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.ListEntries("2026-04-22", "2026-04-22")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len(ListEntries()) = %d, want 2", len(got))
	}
	if got[0].ID != "1" || got[1].ID != "2" {
		t.Fatalf("ListEntries() order = %#v", got)
	}
}

func TestClearEntries(t *testing.T) {
	s := newTestStore(t)
	if err := s.CreateEntry(model.Entry{ID: "1", Timestamp: 100, Date: "2026-04-22", Subject: "math", Minutes: 25}); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearEntries(); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListEntries("2026-04-01", "2026-04-30")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("len(ListEntries()) = %d, want 0", len(got))
	}
}

func TestDeleteEntryByIDNotFound(t *testing.T) {
	s := newTestStore(t)
	if err := s.DeleteEntryByID("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeleteEntryByID() error = %v, want ErrNotFound", err)
	}
}
