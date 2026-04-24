package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type TelegramService struct {
	token      string
	allowedID  int64
	ingest     *IngestService
	httpClient *http.Client
}

type UpdateResponse struct {
	OK          bool     `json:"ok"`
	Result      []Update `json:"result"`
	ErrorCode   int      `json:"error_code"`
	Description string   `json:"description"`
}

type Update struct {
	UpdateID int     `json:"update_id"`
	Message  Message `json:"message"`
}

type Message struct {
	MessageID int64 `json:"message_id"`
	Date      int64 `json:"date"`
	Chat      struct {
		ID int64 `json:"id"`
	} `json:"chat"`
	Text string `json:"text"`
}

func NewTelegramService(token string, allowedID int64, ingest *IngestService) *TelegramService {
	return &TelegramService{
		token:      token,
		allowedID:  allowedID,
		ingest:     ingest,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *TelegramService) Enabled() bool { return strings.TrimSpace(s.token) != "" }

func (s *TelegramService) Run(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}
	offset := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		updates, err := s.getUpdates(ctx, offset)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		for _, update := range updates {
			offset = update.UpdateID + 1
			s.handleMessage(ctx, update.Message)
		}
	}
}

func (s *TelegramService) handleMessage(ctx context.Context, msg Message) {
	if msg.Text == "" {
		return
	}
	if s.allowedID != 0 && msg.Chat.ID != s.allowedID {
		return
	}
	text := strings.TrimSpace(strings.ToLower(msg.Text))
	switch text {
	case "subjects":
		var names []string
		for _, subject := range s.ingest.Subjects() {
			names = append(names, subject.Name)
		}
		s.sendMessage(ctx, msg.Chat.ID, "Subjects: "+strings.Join(names, ", "))
	case "today":
		date := time.Unix(msg.Date, 0).Format(time.DateOnly)
		entries, err := s.ingest.ListEntries(date, date)
		if err != nil {
			s.sendMessage(ctx, msg.Chat.ID, "Could not load today's entries")
			return
		}
		if len(entries) == 0 {
			s.sendMessage(ctx, msg.Chat.ID, "No entries logged today")
			return
		}
		totals := map[string]int{}
		for _, entry := range entries {
			totals[entry.Subject] += entry.Minutes
		}
		var lines []string
		for _, subject := range s.ingest.Subjects() {
			if totals[subject.Name] > 0 {
				lines = append(lines, fmt.Sprintf("%s: %d min", subject.Name, totals[subject.Name]))
			}
		}
		s.sendMessage(ctx, msg.Chat.ID, strings.Join(lines, "\n"))
	case "undo":
		date := time.Unix(msg.Date, 0).Format(time.DateOnly)
		entry, err := s.ingest.Undo(date)
		if err != nil || entry == nil {
			s.sendMessage(ctx, msg.Chat.ID, "Nothing to undo for today")
			return
		}
		s.sendMessage(ctx, msg.Chat.ID, fmt.Sprintf("Removed %d min of %s", entry.Minutes, entry.Subject))
	default:
		entry, err := s.ingest.ParseTelegramMessage(text, time.Unix(msg.Date, 0))
		if err != nil {
			s.sendMessage(ctx, msg.Chat.ID, err.Error())
			return
		}
		s.sendMessage(ctx, msg.Chat.ID, fmt.Sprintf("Logged %d min of %s", entry.Minutes, entry.Subject))
	}
}

func (s *TelegramService) getUpdates(ctx context.Context, offset int) ([]Update, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?timeout=25&offset=%d", s.token, offset)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("telegram getUpdates: unexpected status %s", resp.Status)
	}
	var out UpdateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if !out.OK {
		if out.Description != "" {
			return nil, fmt.Errorf("telegram getUpdates: %s", out.Description)
		}
		if out.ErrorCode != 0 {
			return nil, fmt.Errorf("telegram getUpdates: error code %d", out.ErrorCode)
		}
		return nil, fmt.Errorf("telegram getUpdates: request failed")
	}
	return out.Result, nil
}

func (s *TelegramService) sendMessage(ctx context.Context, chatID int64, text string) {
	body, _ := json.Marshal(map[string]any{"chat_id": chatID, "text": text})
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.token)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err == nil && resp != nil {
		resp.Body.Close()
	}
}
