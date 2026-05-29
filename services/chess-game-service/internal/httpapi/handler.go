package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"example.com/chess-platform/internal/platform"
	game "example.com/chess-platform/services/chess-game-service/internal/service"
)

type Handler struct {
	stats   *platform.StatsCollector
	service *game.Service
}

// NewHandler связывает HTTP-слой с бизнес-логикой сервиса партий.
func NewHandler(stats *platform.StatsCollector, service *game.Service) *Handler {
	return &Handler{
		stats:   stats,
		service: service,
	}
}

// Routes описывает все доступные endpoints сервиса партий.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /stats", h.statsEndpoint)
	mux.HandleFunc("POST /api/v1/games", h.createGame)
	mux.HandleFunc("GET /api/v1/games/{id}", h.getGame)
	mux.HandleFunc("POST /api/v1/games/{id}/resign", h.resignGame)

	return mux
}

// health — техническая проверка, что процесс отвечает по HTTP.
func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	platform.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// statsEndpoint отдает служебную информацию о процессе.
func (h *Handler) statsEndpoint(w http.ResponseWriter, _ *http.Request) {
	platform.WriteJSON(w, http.StatusOK, h.stats.Snapshot())
}

// createGame принимает запрос на создание новой партии.
func (h *Handler) createGame(w http.ResponseWriter, r *http.Request) {
	var input game.CreateGameRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "invalid json body")
		return
	}

	created, err := h.service.CreateGame(input)
	if err != nil {
		if errors.Is(err, game.ErrInvalidPlayers) {
			platform.WriteError(w, http.StatusBadRequest, "invalid_players", err.Error())
			return
		}

		platform.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to create game")
		return
	}

	platform.WriteJSON(w, http.StatusCreated, created)
}

// getGame возвращает текущее состояние партии.
func (h *Handler) getGame(w http.ResponseWriter, r *http.Request) {
	found, err := h.service.GetGame(r.PathValue("id"))
	if err != nil {
		if errors.Is(err, game.ErrGameNotFound) {
			platform.WriteError(w, http.StatusNotFound, "game_not_found", err.Error())
			return
		}

		platform.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load game")
		return
	}

	platform.WriteJSON(w, http.StatusOK, found)
}

// resignGame завершает партию по сдаче игрока.
func (h *Handler) resignGame(w http.ResponseWriter, r *http.Request) {
	var input game.ResignRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "invalid json body")
		return
	}

	updated, err := h.service.ResignGame(r.PathValue("id"), input)
	if err != nil {
		switch {
		case errors.Is(err, game.ErrGameNotFound):
			platform.WriteError(w, http.StatusNotFound, "game_not_found", err.Error())
		case errors.Is(err, game.ErrGameFinished):
			platform.WriteError(w, http.StatusConflict, "game_finished", err.Error())
		case errors.Is(err, game.ErrInvalidColor):
			platform.WriteError(w, http.StatusBadRequest, "invalid_color", err.Error())
		default:
			platform.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to resign game")
		}
		return
	}

	platform.WriteJSON(w, http.StatusOK, updated)
}
