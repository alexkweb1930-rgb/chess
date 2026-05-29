package platform

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type StatsSnapshot struct {
	ServiceName   string            `json:"serviceName"`
	Version       string            `json:"version"`
	StartedAt     time.Time         `json:"startedAt"`
	UptimeSeconds int64             `json:"uptimeSeconds"`
	TotalRequests uint64            `json:"totalRequests"`
	ResponseCodes map[string]uint64 `json:"responseCodes"`
}

// StatsCollector хранит сервисную статистику процесса:
// когда сервис стартовал, сколько запросов обработал и с какими кодами ответа.
type StatsCollector struct {
	serviceName string
	version     string
	startedAt   time.Time

	totalRequests atomic.Uint64

	mu            sync.RWMutex
	responseCodes map[int]uint64
}

// NewStatsCollector создается один раз при старте сервиса.
func NewStatsCollector(serviceName, version string) *StatsCollector {
	return &StatsCollector{
		serviceName:   serviceName,
		version:       version,
		startedAt:     time.Now().UTC(),
		responseCodes: make(map[int]uint64),
	}
}

// Record вызывается middleware после каждого HTTP-запроса.
func (s *StatsCollector) Record(statusCode int) {
	s.totalRequests.Add(1)

	s.mu.Lock()
	s.responseCodes[statusCode]++
	s.mu.Unlock()
}

// Snapshot готовит данные для endpoint /stats.
func (s *StatsCollector) Snapshot() StatsSnapshot {
	s.mu.RLock()
	responseCodes := make(map[string]uint64, len(s.responseCodes))
	for code, count := range s.responseCodes {
		responseCodes[strconv.Itoa(code)] = count
	}
	s.mu.RUnlock()

	return StatsSnapshot{
		ServiceName:   s.serviceName,
		Version:       s.version,
		StartedAt:     s.startedAt,
		UptimeSeconds: int64(time.Since(s.startedAt).Seconds()),
		TotalRequests: s.totalRequests.Load(),
		ResponseCodes: responseCodes,
	}
}

// NewLogger поднимает JSON-логгер стандартной библиотеки.
// Этого достаточно для локального запуска, Docker и дальнейшей отправки логов в Elastic.
func NewLogger(level string) *slog.Logger {
	var slogLevel slog.Level

	switch strings.ToLower(level) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slogLevel,
	}))
}
