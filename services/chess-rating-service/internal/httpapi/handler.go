package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"example.com/chess-platform/internal/platform"
	rating "example.com/chess-platform/services/chess-rating-service/internal/service"
)

type Handler struct {
	stats   *platform.StatsCollector
	service *rating.Service
}

// NewHandler связывает HTTP endpoints с логикой рейтинг-сервиса.
func NewHandler(stats *platform.StatsCollector, service *rating.Service) *Handler {
	return &Handler{
		stats:   stats,
		service: service,
	}
}

// Routes описывает внешние и внутренние endpoints рейтинг-сервиса.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /stats", h.statsEndpoint)
	mux.HandleFunc("POST /api/v1/ratings/players", h.createProfile)
	mux.HandleFunc("GET /api/v1/ratings/players/{playerId}", h.getProfile)
	mux.HandleFunc("GET /api/v1/ratings/leaderboard", h.leaderboard)
	mux.HandleFunc("POST /internal/v1/rating/game-result", h.applyGameResult)

	return mux
}

// health используется для проверки доступности сервиса.
func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	platform.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// statsEndpoint отдает техническую статистику процесса.
func (h *Handler) statsEndpoint(w http.ResponseWriter, _ *http.Request) {
	platform.WriteJSON(w, http.StatusOK, h.stats.Snapshot())
}

// createProfile создает рейтинг-профиль игрока через публичный API.
func (h *Handler) createProfile(w http.ResponseWriter, r *http.Request) {
	var input rating.CreateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "invalid json body")
		return
	}

	profile, err := h.service.CreateProfile(input)
	if err != nil {
		switch {
		case errors.Is(err, rating.ErrPlayerIDRequired):
			platform.WriteError(w, http.StatusBadRequest, "player_id_required", err.Error())
		case errors.Is(err, rating.ErrProfileExists):
			platform.WriteError(w, http.StatusConflict, "profile_exists", err.Error())
		default:
			platform.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to create profile")
		}
		return
	}

	platform.WriteJSON(w, http.StatusCreated, profile)
}

// getProfile возвращает рейтинг и статистику конкретного игрока.
func (h *Handler) getProfile(w http.ResponseWriter, r *http.Request) {
	profile, err := h.service.GetProfile(r.PathValue("playerId"))
	if err != nil {
		if errors.Is(err, rating.ErrProfileNotFound) {
			platform.WriteError(w, http.StatusNotFound, "profile_not_found", err.Error())
			return
		}

		platform.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load profile")
		return
	}

	platform.WriteJSON(w, http.StatusOK, profile)
}

// leaderboard возвращает игроков, отсортированных по рейтингу.
func (h *Handler) leaderboard(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil {
			limit = parsed
		}
	}

	platform.WriteJSON(w, http.StatusOK, map[string]any{
		"items": h.service.Leaderboard(limit),
	})
}

// applyGameResult — внутренний endpoint, который дергает сервис партий
// после завершения игры, чтобы обновить рейтинги игроков.
func (h *Handler) applyGameResult(w http.ResponseWriter, r *http.Request) {
	var input rating.ApplyGameResultRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "invalid json body")
		return
	}

	result, err := h.service.ApplyGameResult(input)
	if err != nil {
		if errors.Is(err, rating.ErrInvalidResult) || errors.Is(err, rating.ErrInvalidGameData) {
			platform.WriteError(w, http.StatusBadRequest, "invalid_result", err.Error())
			return
		}

		platform.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to apply game result")
		return
	}

	platform.WriteJSON(w, http.StatusOK, result)
}
