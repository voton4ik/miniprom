package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
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

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("конфигурация: %v", err)
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
		log.Printf("miniprom слушает на %s", cfg.ListenAddr)
		log.Printf("интервал сбора: %s, хранение: %s, целей: %d",
			cfg.ScrapeInterval.Duration, cfg.Retention.Duration, len(cfg.Targets))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("сервер: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("получен сигнал завершения, останавливаюсь...")
	close(stopCompaction)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("ошибка остановки: %v", err)
		os.Exit(1)
	}
	log.Println("остановлено")
}
