package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.etcd.io/bbolt"

	"study-blocks/internal/model"
)

var (
	entriesBucket  = []byte("entries")
	metadataBucket = []byte("metadata")
	ErrNotFound    = errors.New("not found")
)

type Store struct{ db *bbolt.DB }

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir db dir: %w", err)
	}
	db, err := bbolt.Open(path, 0o600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("open boltdb %q: %w", path, err)
	}
	if err := db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(entriesBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(metadataBucket); err != nil {
			return err
		}
		return nil
	}); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) CreateEntry(entry model.Entry) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(entriesBucket)
		buf, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		return b.Put([]byte(entryKey(entry)), buf)
	})
}

func (s *Store) ListEntries(from, to string) ([]model.Entry, error) {
	var entries []model.Entry
	err := s.db.View(func(tx *bbolt.Tx) error {
		c := tx.Bucket(entriesBucket).Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			date := keyDate(string(k))
			if date < from || date > to {
				continue
			}
			var entry model.Entry
			if err := json.Unmarshal(v, &entry); err != nil {
				return err
			}
			entries = append(entries, entry)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Date == entries[j].Date {
			return entries[i].Timestamp < entries[j].Timestamp
		}
		return entries[i].Date < entries[j].Date
	})
	return entries, nil
}

func (s *Store) DeleteEntryByID(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		c := tx.Bucket(entriesBucket).Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var entry model.Entry
			if err := json.Unmarshal(v, &entry); err != nil {
				return err
			}
			if entry.ID == id {
				return c.Delete()
			}
		}
		return ErrNotFound
	})
}

func (s *Store) DeleteLatestEntryForDate(date string) (*model.Entry, error) {
	var latest *model.Entry
	var latestKey []byte
	err := s.db.Update(func(tx *bbolt.Tx) error {
		c := tx.Bucket(entriesBucket).Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if keyDate(string(k)) != date {
				continue
			}
			var entry model.Entry
			if err := json.Unmarshal(v, &entry); err != nil {
				return err
			}
			if latest == nil || entry.Timestamp >= latest.Timestamp {
				copyEntry := entry
				latest = &copyEntry
				latestKey = append([]byte(nil), k...)
			}
		}
		if latest == nil {
			return nil
		}
		return tx.Bucket(entriesBucket).Delete(latestKey)
	})
	return latest, err
}

func (s *Store) ClearEntries() error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		if err := tx.DeleteBucket(entriesBucket); err != nil && !errors.Is(err, bbolt.ErrBucketNotFound) {
			return err
		}
		_, err := tx.CreateBucketIfNotExists(entriesBucket)
		return err
	})
}

func entryKey(entry model.Entry) string {
	return fmt.Sprintf("%s:%020d:%s", entry.Date, entry.Timestamp, entry.ID)
}

func keyDate(key string) string {
	parts := strings.SplitN(key, ":", 2)
	return parts[0]
}
