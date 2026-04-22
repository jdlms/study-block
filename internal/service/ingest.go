package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"study-blocks/internal/model"
	"study-blocks/internal/store"
)

type IngestService struct {
	store    *store.Store
	subjects []model.Subject
	allowed  map[string]struct{}
}

func NewIngestService(store *store.Store, subjects []model.Subject) *IngestService {
	allowed := make(map[string]struct{}, len(subjects))
	for _, subject := range subjects {
		allowed[subject.Name] = struct{}{}
	}
	return &IngestService{store: store, subjects: subjects, allowed: allowed}
}

func (s *IngestService) Subjects() []model.Subject {
	return append([]model.Subject(nil), s.subjects...)
}

func (s *IngestService) CreateEntry(date, subject string, minutes int, ts time.Time) (model.Entry, error) {
	subject = strings.ToLower(strings.TrimSpace(subject))
	if _, ok := s.allowed[subject]; !ok {
		return model.Entry{}, fmt.Errorf("unknown subject: %s", subject)
	}
	if minutes <= 0 {
		return model.Entry{}, fmt.Errorf("minutes must be positive")
	}
	if date == "" {
		date = ts.Format(time.DateOnly)
	}
	entry := model.Entry{
		ID:        newID(),
		Timestamp: ts.Unix(),
		Date:      date,
		Subject:   subject,
		Minutes:   minutes,
	}
	if err := s.store.CreateEntry(entry); err != nil {
		return model.Entry{}, err
	}
	return entry, nil
}

func (s *IngestService) ParseTelegramMessage(text string, ts time.Time) (model.Entry, error) {
	parts := strings.Fields(strings.TrimSpace(text))
	if len(parts) != 2 {
		return model.Entry{}, fmt.Errorf("expected '<minutes> <subject>'")
	}
	minutes, err := strconv.Atoi(parts[0])
	if err != nil || minutes <= 0 {
		return model.Entry{}, fmt.Errorf("minutes must be a positive integer")
	}
	return s.CreateEntry(ts.Format(time.DateOnly), parts[1], minutes, ts)
}

func (s *IngestService) ListEntries(from, to string) ([]model.Entry, error) {
	return s.store.ListEntries(from, to)
}
func (s *IngestService) DeleteEntry(id string) error { return s.store.DeleteEntryByID(id) }
func (s *IngestService) Undo(date string) (*model.Entry, error) {
	return s.store.DeleteLatestEntryForDate(date)
}
func (s *IngestService) ClearEntries() error { return s.store.ClearEntries() }

func newID() string {
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}
