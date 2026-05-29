package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"example.com/chess-platform/internal/platform"
	"example.com/chess-platform/services/chess-rating-service/internal/config"
	"example.com/chess-platform/services/chess-rating-service/internal/httpapi"
	rating "example.com/chess-platform/services/chess-rating-service/internal/service"
)

func main() {
	// Загружаем конфиг именно рейтинг-сервиса.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Инициализация остается линейной и простой:
	// конфиг -> логгер/stats -> сервис -> handlers -> HTTP server.
	logger := platform.NewLogger(cfg.LogLevel)
	stats := platform.NewStatsCollector(cfg.ServiceName, cfg.Version)
	ratingService := rating.NewService(cfg.InitialRating, cfg.KFactor)
	handler := httpapi.NewHandler(stats, ratingService)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      platform.WithCORS(platform.AccessLog(logger, stats)(handler.Routes())),
		ReadTimeout:  time.Duration(cfg.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeoutSeconds) * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		// Отдельная goroutine позволяет параллельно слушать сигнал завершения.
		logger.Info("starting rating service", "port", cfg.Port, "initial_rating", cfg.InitialRating, "k_factor", cfg.KFactor)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("rating service failed: %v", err)
		}
	case <-ctx.Done():
		// Graceful shutdown нужен, чтобы сервис корректно завершался в Docker и CI/CD.
		logger.Info("shutdown signal received")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownTimeoutSeconds)*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("shutdown rating service: %v", err)
		}
	}
}
