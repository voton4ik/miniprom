package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/voton4ik/miniprom/internal/api"
	"github.com/voton4ik/miniprom/internal/config"
	"github.com/voton4ik/miniprom/internal/scraper"
	"github.com/voton4ik/miniprom/internal/storage"
)

func main() {
	configPath := flag.String("config", "config.json", "путь к файлу конфигурации")
	flag.Parse()

	setupLogger(os.Getenv("MINIPROM_LOG_LEVEL"))

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("не удалось загрузить конфигурацию", "error", err)
		os.Exit(1)
	}

	store := storage.New(cfg.Retention.Duration)
	manager := scraper.New(store, cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	stopCompaction := make(chan struct{})
	go store.RunCompaction(stopCompaction, cfg.ScrapeInterval.Duration)
	manager.Start(ctx)

	srv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: api.New(store, manager).Handler(),
	}

	go func() {
		slog.Info("miniprom запущен",
			"addr", cfg.ListenAddr,
			"scrape_interval", cfg.ScrapeInterval.Duration,
			"retention", cfg.Retention.Duration,
			"targets", len(cfg.Targets))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("сбой http-сервера", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("получен сигнал завершения, останавливаюсь")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("http-сервер не остановился штатно", "error", err)
	}

	close(stopCompaction)
	manager.Wait()
	slog.Info("остановлено")
}

func setupLogger(level string) {
	lvl := slog.LevelInfo
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	}
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(handler))
}
