package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"study-blocks/internal/model"
	"study-blocks/internal/store"
)

type IngestService struct {
	store    *store.Store
	subjects []model.Subject
	allowed  map[string]struct{}
	mu       sync.RWMutex
}

var hexColorPattern = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

func NewIngestService(store *store.Store, subjects []model.Subject) *IngestService {
	copySubjects := append([]model.Subject(nil), subjects...)
	allowed := make(map[string]struct{}, len(copySubjects))
	for _, subject := range copySubjects {
		allowed[subject.Name] = struct{}{}
	}
	return &IngestService{store: store, subjects: copySubjects, allowed: allowed}
}

func (s *IngestService) Subjects() []model.Subject {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]model.Subject(nil), s.subjects...)
}

func (s *IngestService) CreateEntry(date, subject string, minutes int, ts time.Time) (model.Entry, error) {
	subject = normalizeSubjectName(subject)
	s.mu.RLock()
	_, ok := s.allowed[subject]
	s.mu.RUnlock()
	if !ok {
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
		return model.Entry{}, fmt.Errorf("expected '<minutes> <subject>' or '<subject> <minutes>'")
	}

	var minutes int
	var subject string

	if m, err := strconv.Atoi(parts[0]); err == nil && m > 0 {
		minutes = m
		subject = parts[1]
	} else if m, err := strconv.Atoi(parts[1]); err == nil && m > 0 {
		minutes = m
		subject = parts[0]
	} else {
		return model.Entry{}, fmt.Errorf("one part must be a positive number of minutes")
	}

	return s.CreateEntry(ts.Format(time.DateOnly), subject, minutes, ts)
}

func (s *IngestService) AddSubject(name string) (model.Subject, error) {
	name = normalizeSubjectName(name)
	if name == "" {
		return model.Subject{}, fmt.Errorf("subject name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.allowed[name]; ok {
		return model.Subject{}, fmt.Errorf("subject already exists: %s", name)
	}
	subject := model.Subject{Name: name, Color: model.DefaultSubjectColor(len(s.subjects))}
	next := append(append([]model.Subject(nil), s.subjects...), subject)
	if err := s.store.SaveSubjects(next); err != nil {
		return model.Subject{}, err
	}
	s.subjects = next
	s.allowed[name] = struct{}{}
	return subject, nil
}

func (s *IngestService) UpdateSubjectColor(name, color string) (model.Subject, error) {
	name = normalizeSubjectName(name)
	color = strings.TrimSpace(color)
	if !hexColorPattern.MatchString(color) {
		return model.Subject{}, fmt.Errorf("color must be #RGB or #RRGGBB")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i, subject := range s.subjects {
		if subject.Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return model.Subject{}, fmt.Errorf("unknown subject: %s", name)
	}
	next := append([]model.Subject(nil), s.subjects...)
	next[idx].Color = color
	if err := s.store.SaveSubjects(next); err != nil {
		return model.Subject{}, err
	}
	s.subjects = next
	return next[idx], nil
}

func (s *IngestService) ListEntries(from, to string) ([]model.Entry, error) {
	return s.store.ListEntries(from, to)
}
func (s *IngestService) DeleteEntry(id string) error { return s.store.DeleteEntryByID(id) }
func (s *IngestService) Undo(date string) (*model.Entry, error) {
	return s.store.DeleteLatestEntryForDate(date)
}
func (s *IngestService) ClearEntries() error { return s.store.ClearEntries() }

func normalizeSubjectName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func newID() string {
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}
