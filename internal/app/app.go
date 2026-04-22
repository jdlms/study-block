package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"study-blocks/internal/config"
	"study-blocks/internal/handler"
	"study-blocks/internal/model"
	"study-blocks/internal/service"
	"study-blocks/internal/store"
)

type App struct {
	server   *http.Server
	store    *store.Store
	telegram *service.TelegramService
	httpAddr string
	localURL string
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	db, err := store.Open(cfg.BoltDBPath)
	if err != nil {
		return nil, err
	}
	subjects := model.SubjectsFromNames(cfg.Subjects)
	ingest := service.NewIngestService(db, subjects)
	api := handler.NewAPIHandler(ingest, cfg.EnableLocalTestRoutes)
	mux := http.NewServeMux()
	api.Register(mux)
	if err := handler.RegisterFrontend(mux); err != nil {
		return nil, err
	}
	return &App{
		server: &http.Server{
			Addr:              cfg.HTTPAddr,
			Handler:           loggingMiddleware(mux),
			ReadHeaderTimeout: 5 * time.Second,
		},
		store:    db,
		telegram: service.NewTelegramService(cfg.TelegramBotToken, cfg.TelegramAllowedChatID, ingest),
		httpAddr: cfg.HTTPAddr,
		localURL: localURL(cfg.HTTPAddr),
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 2)
	log.Printf("study-blocks listening on %s", a.httpAddr)
	log.Printf("frontend: %s", a.localURL)
	log.Printf("health:   %s/api/health", a.localURL)

	go func() {
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	go func() {
		if err := a.telegram.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- err
		}
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return a.server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func (a *App) Close() error {
	if a.store == nil {
		return nil
	}
	return a.store.Close()
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(sw, r)
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, sw.status, fmtDuration(time.Since(start)))
	})
}

func fmtDuration(d time.Duration) string { return fmt.Sprintf("%dms", d.Milliseconds()) }

func localURL(addr string) string {
	if addr == "" || addr == ":8080" {
		return "http://localhost:8080"
	}
	if addr[0] == ':' {
		return "http://localhost" + addr
	}
	return "http://" + addr
}
