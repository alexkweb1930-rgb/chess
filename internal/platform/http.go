package platform

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader перехватывает HTTP-код ответа, чтобы потом отдать его в stats и логирование.
func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Write нужен на случай, когда handler пишет тело ответа без явного WriteHeader.
func (r *statusRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}

	return r.ResponseWriter.Write(data)
}

// AccessLog — общий middleware для обоих сервисов.
// Он считает запросы, сохраняет коды ответов и пишет короткий structured log.
func AccessLog(logger *slog.Logger, stats *StatsCollector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			recorder := &statusRecorder{ResponseWriter: w}

			next.ServeHTTP(recorder, r)

			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}

			stats.Record(status)

			logger.Info(
				"request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"duration_ms", time.Since(startedAt).Milliseconds(),
			)
		})
	}
}

// WithCORS добавляет минимальные CORS-заголовки, чтобы простой фронт мог
// напрямую обращаться к обоим сервисам из браузера.
func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// WriteJSON — единая точка для JSON-ответов, чтобы не дублировать одинаковый код в handlers.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, `{"error":{"code":"encode_error","message":"failed to encode response"}}`, http.StatusInternalServerError)
	}
}

// WriteError отдает ошибку в едином формате API.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
