package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddr              string
	BoltDBPath            string
	Subjects              []string
	TelegramBotToken      string
	TelegramAllowedChatID int64
	EnableLocalTestRoutes bool
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:              envOrDefault("HTTP_ADDR", ":8080"),
		BoltDBPath:            envOrDefault("BOLTDB_PATH", "data/study-blocks.db"),
		TelegramBotToken:      strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")),
		EnableLocalTestRoutes: parseBoolEnv("ENABLE_LOCAL_TEST_ROUTES"),
	}

	subjects, err := parseSubjects(os.Getenv("SUBJECTS"))
	if err != nil {
		return Config{}, err
	}
	cfg.Subjects = subjects

	allowed := envOrDefault("TELEGRAM_ALLOWED_CHAT_ID", "0")
	chatID, err := strconv.ParseInt(strings.TrimSpace(allowed), 10, 64)
	if err != nil {
		return Config{}, fmt.Errorf("parse TELEGRAM_ALLOWED_CHAT_ID: %w", err)
	}
	cfg.TelegramAllowedChatID = chatID

	return cfg, nil
}

func parseSubjects(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	seen := map[string]struct{}{}
	var out []string
	for _, part := range parts {
		s := strings.ToLower(strings.TrimSpace(part))
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("SUBJECTS is required")
	}
	return out, nil
}

func envOrDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func parseBoolEnv(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
