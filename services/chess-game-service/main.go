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
	"example.com/chess-platform/services/chess-game-service/internal/config"
	"example.com/chess-platform/services/chess-game-service/internal/httpapi"
	"example.com/chess-platform/services/chess-game-service/internal/ratingclient"
	game "example.com/chess-platform/services/chess-game-service/internal/service"
)

func main() {
	// Загружаем конфиг именно сервиса партий.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Собираем зависимости максимально прямо:
	// логгер, stats, HTTP-клиент до рейтинг-сервиса, бизнес-логика и HTTP-handlers.
	logger := platform.NewLogger(cfg.LogLevel)
	stats := platform.NewStatsCollector(cfg.ServiceName, cfg.Version)
	ratingClient := ratingclient.New(cfg.RatingServiceBaseURL, time.Duration(cfg.RatingServiceTimeoutSeconds)*time.Second)
	gameService := game.NewService(ratingClient)
	handler := httpapi.NewHandler(stats, gameService)

	// Поднимаем обычный net/http сервер без дополнительного фреймворка.
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
		// Сервис стартует в отдельной goroutine, а основной поток ждет либо ошибку, либо сигнал остановки.
		logger.Info("starting game service", "port", cfg.Port, "rating_service_url", cfg.RatingServiceBaseURL)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("game service failed: %v", err)
		}
	case <-ctx.Done():
		// Даем серверу время аккуратно завершить активные запросы.
		logger.Info("shutdown signal received")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownTimeoutSeconds)*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("shutdown game service: %v", err)
		}
	}
}
