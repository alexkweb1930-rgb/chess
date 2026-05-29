package game

import (
	"errors"
	"sync"
	"time"

	"example.com/chess-platform/internal/platform"
	"example.com/chess-platform/services/chess-game-service/internal/ratingclient"
)

type Status string

const (
	StatusInProgress Status = "in_progress"
	StatusResigned   Status = "resigned"
)

type Result string

const (
	ResultNone     Result = "none"
	ResultWhiteWin Result = "white_win"
	ResultBlackWin Result = "black_win"
)

type PlayerColor string

const (
	PlayerWhite PlayerColor = "white"
	PlayerBlack PlayerColor = "black"
)

type Game struct {
	ID                 string      `json:"id"`
	WhitePlayerID      string      `json:"whitePlayerId"`
	BlackPlayerID      string      `json:"blackPlayerId"`
	Status             Status      `json:"status"`
	Result             Result      `json:"result"`
	CurrentTurn        PlayerColor `json:"currentTurn"`
	BoardFEN           string      `json:"boardFEN"`
	RatingUpdateStatus string      `json:"ratingUpdateStatus"`
	CreatedAt          time.Time   `json:"createdAt"`
	UpdatedAt          time.Time   `json:"updatedAt"`
	FinishedAt         *time.Time  `json:"finishedAt,omitempty"`
}

type CreateGameRequest struct {
	WhitePlayerID string `json:"whitePlayerId"`
	BlackPlayerID string `json:"blackPlayerId"`
}

type ResignRequest struct {
	PlayerColor PlayerColor `json:"playerColor"`
}

var (
	ErrInvalidPlayers = errors.New("whitePlayerId and blackPlayerId are required")
	ErrGameNotFound   = errors.New("game not found")
	ErrGameFinished   = errors.New("game already finished")
	ErrInvalidColor   = errors.New("playerColor must be white or black")
)

// Service — простое in-memory хранилище партий для MVP.
// Здесь нет репозитория и БД, чтобы код оставался коротким и прозрачным.
type Service struct {
	mu           sync.RWMutex
	games        map[string]*Game
	ratingClient *ratingclient.Client
}

// NewService создает сервис партий и получает клиента для связи с рейтинг-сервисом.
func NewService(ratingClient *ratingclient.Client) *Service {
	return &Service{
		games:        make(map[string]*Game),
		ratingClient: ratingClient,
	}
}

// CreateGame создает новую партию в стартовом состоянии.
func (s *Service) CreateGame(input CreateGameRequest) (*Game, error) {
	if input.WhitePlayerID == "" || input.BlackPlayerID == "" {
		return nil, ErrInvalidPlayers
	}

	now := time.Now().UTC()
	game := &Game{
		ID:                 platform.NewID(),
		WhitePlayerID:      input.WhitePlayerID,
		BlackPlayerID:      input.BlackPlayerID,
		Status:             StatusInProgress,
		Result:             ResultNone,
		CurrentTurn:        PlayerWhite,
		BoardFEN:           "startpos",
		RatingUpdateStatus: "not_required",
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	s.mu.Lock()
	s.games[game.ID] = game
	s.mu.Unlock()

	return cloneGame(game), nil
}

// GetGame возвращает снимок партии по id.
func (s *Service) GetGame(id string) (*Game, error) {
	s.mu.RLock()
	game, ok := s.games[id]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrGameNotFound
	}

	return cloneGame(game), nil
}

// ResignGame завершает партию по сдаче и сразу пытается отправить
// результат в рейтинг-сервис. Если внешний вызов не удался, сама партия
// все равно остается завершенной, а статус отправки помечается как failed.
func (s *Service) ResignGame(id string, input ResignRequest) (*Game, error) {
	if input.PlayerColor != PlayerWhite && input.PlayerColor != PlayerBlack {
		return nil, ErrInvalidColor
	}

	s.mu.Lock()
	game, ok := s.games[id]
	if !ok {
		s.mu.Unlock()
		return nil, ErrGameNotFound
	}
	if game.Status == StatusResigned {
		s.mu.Unlock()
		return nil, ErrGameFinished
	}

	now := time.Now().UTC()
	game.Status = StatusResigned
	game.UpdatedAt = now
	game.FinishedAt = &now
	game.RatingUpdateStatus = "pending"

	if input.PlayerColor == PlayerWhite {
		game.Result = ResultBlackWin
	} else {
		game.Result = ResultWhiteWin
	}

	payload := ratingclient.GameResultRequest{
		GameID:        game.ID,
		WhitePlayerID: game.WhitePlayerID,
		BlackPlayerID: game.BlackPlayerID,
		Result:        string(game.Result),
		FinishedAt:    now,
	}
	s.mu.Unlock()

	err := s.ratingClient.SendGameResult(payload)

	s.mu.Lock()
	defer s.mu.Unlock()

	current := s.games[id]
	if current == nil {
		return nil, ErrGameNotFound
	}

	if err != nil {
		current.RatingUpdateStatus = "failed"
		return cloneGame(current), nil
	}

	current.RatingUpdateStatus = "sent"
	return cloneGame(current), nil
}

// cloneGame защищает внутреннее состояние от случайного изменения снаружи.
func cloneGame(source *Game) *Game {
	if source == nil {
		return nil
	}

	clone := *source
	if source.FinishedAt != nil {
		finishedAt := *source.FinishedAt
		clone.FinishedAt = &finishedAt
	}

	return &clone
}
