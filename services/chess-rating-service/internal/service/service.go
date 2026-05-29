package rating

import (
	"errors"
	"math"
	"sort"
	"sync"
	"time"

	"example.com/chess-platform/internal/platform"
)

type Profile struct {
	PlayerID    string    `json:"playerId"`
	Rating      int       `json:"rating"`
	GamesPlayed int       `json:"gamesPlayed"`
	Wins        int       `json:"wins"`
	Losses      int       `json:"losses"`
	Draws       int       `json:"draws"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type HistoryEntry struct {
	ID           string    `json:"id"`
	PlayerID     string    `json:"playerId"`
	GameID       string    `json:"gameId"`
	RatingBefore int       `json:"ratingBefore"`
	RatingAfter  int       `json:"ratingAfter"`
	RatingDelta  int       `json:"ratingDelta"`
	Result       string    `json:"result"`
	CreatedAt    time.Time `json:"createdAt"`
}

type CreateProfileRequest struct {
	PlayerID      string `json:"playerId"`
	InitialRating int    `json:"initialRating"`
}

type ApplyGameResultRequest struct {
	GameID        string    `json:"gameId"`
	WhitePlayerID string    `json:"whitePlayerId"`
	BlackPlayerID string    `json:"blackPlayerId"`
	Result        string    `json:"result"`
	FinishedAt    time.Time `json:"finishedAt"`
}

type ApplyGameResultResponse struct {
	WhitePlayerID     string `json:"whitePlayerId"`
	WhiteRatingBefore int    `json:"whiteRatingBefore"`
	WhiteRatingAfter  int    `json:"whiteRatingAfter"`
	BlackPlayerID     string `json:"blackPlayerId"`
	BlackRatingBefore int    `json:"blackRatingBefore"`
	BlackRatingAfter  int    `json:"blackRatingAfter"`
}

var (
	ErrPlayerIDRequired = errors.New("playerId is required")
	ErrProfileExists    = errors.New("rating profile already exists")
	ErrProfileNotFound  = errors.New("rating profile not found")
	ErrInvalidResult    = errors.New("result must be white_win, black_win or draw")
	ErrInvalidGameData  = errors.New("gameId, whitePlayerId and blackPlayerId are required")
)

// Service хранит рейтинги и историю изменений в памяти.
type Service struct {
	mu            sync.RWMutex
	initialRating int
	kFactor       int
	profiles      map[string]*Profile
	history       map[string][]HistoryEntry
}

// NewService создает сервис рейтингов с параметрами Elo из конфига.
func NewService(initialRating, kFactor int) *Service {
	return &Service{
		initialRating: initialRating,
		kFactor:       kFactor,
		profiles:      make(map[string]*Profile),
		history:       make(map[string][]HistoryEntry),
	}
}

// CreateProfile создает профиль игрока вручную через публичный API.
func (s *Service) CreateProfile(input CreateProfileRequest) (*Profile, error) {
	if input.PlayerID == "" {
		return nil, ErrPlayerIDRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.profiles[input.PlayerID]; exists {
		return nil, ErrProfileExists
	}

	initialRating := input.InitialRating
	if initialRating == 0 {
		initialRating = s.initialRating
	}

	now := time.Now().UTC()
	profile := &Profile{
		PlayerID:    input.PlayerID,
		Rating:      initialRating,
		GamesPlayed: 0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.profiles[input.PlayerID] = profile
	return cloneProfile(profile), nil
}

// GetProfile возвращает текущий рейтинг и счетчики игрока.
func (s *Service) GetProfile(playerID string) (*Profile, error) {
	s.mu.RLock()
	profile, ok := s.profiles[playerID]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrProfileNotFound
	}

	return cloneProfile(profile), nil
}

// Leaderboard строит простой рейтинг игроков по убыванию Elo.
func (s *Service) Leaderboard(limit int) []Profile {
	if limit <= 0 {
		limit = 10
	}

	s.mu.RLock()
	items := make([]Profile, 0, len(s.profiles))
	for _, profile := range s.profiles {
		items = append(items, *cloneProfile(profile))
	}
	s.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		if items[i].Rating == items[j].Rating {
			return items[i].PlayerID < items[j].PlayerID
		}
		return items[i].Rating > items[j].Rating
	})

	if limit > len(items) {
		limit = len(items)
	}

	return items[:limit]
}

// ApplyGameResult — главная бизнес-функция рейтинг-сервиса.
// Она принимает завершенную партию, считает новый Elo для обоих игроков
// и обновляет статистику побед, поражений и ничьих.
func (s *Service) ApplyGameResult(input ApplyGameResultRequest) (*ApplyGameResultResponse, error) {
	if input.GameID == "" || input.WhitePlayerID == "" || input.BlackPlayerID == "" {
		return nil, ErrInvalidGameData
	}
	if input.Result != "white_win" && input.Result != "black_win" && input.Result != "draw" {
		return nil, ErrInvalidResult
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	white := s.ensureProfile(input.WhitePlayerID)
	black := s.ensureProfile(input.BlackPlayerID)

	whiteBefore := white.Rating
	blackBefore := black.Rating

	whiteActual, blackActual := actualScores(input.Result)
	whiteExpected := expectedScore(whiteBefore, blackBefore)
	blackExpected := expectedScore(blackBefore, whiteBefore)

	whiteAfter := recalculateRating(whiteBefore, s.kFactor, whiteActual, whiteExpected)
	blackAfter := recalculateRating(blackBefore, s.kFactor, blackActual, blackExpected)

	white.Rating = whiteAfter
	black.Rating = blackAfter
	white.GamesPlayed++
	black.GamesPlayed++
	white.UpdatedAt = time.Now().UTC()
	black.UpdatedAt = white.UpdatedAt

	switch input.Result {
	case "white_win":
		white.Wins++
		black.Losses++
	case "black_win":
		black.Wins++
		white.Losses++
	case "draw":
		white.Draws++
		black.Draws++
	}

	s.appendHistory(white.PlayerID, input.GameID, whiteBefore, whiteAfter, playerResult("white", input.Result), input.FinishedAt)
	s.appendHistory(black.PlayerID, input.GameID, blackBefore, blackAfter, playerResult("black", input.Result), input.FinishedAt)

	return &ApplyGameResultResponse{
		WhitePlayerID:     white.PlayerID,
		WhiteRatingBefore: whiteBefore,
		WhiteRatingAfter:  whiteAfter,
		BlackPlayerID:     black.PlayerID,
		BlackRatingBefore: blackBefore,
		BlackRatingAfter:  blackAfter,
	}, nil
}

// ensureProfile автоматически создает профиль, если игрок пришел впервые.
func (s *Service) ensureProfile(playerID string) *Profile {
	if profile, ok := s.profiles[playerID]; ok {
		return profile
	}

	now := time.Now().UTC()
	profile := &Profile{
		PlayerID:  playerID,
		Rating:    s.initialRating,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.profiles[playerID] = profile
	return profile
}

// appendHistory сохраняет изменение рейтинга после конкретной партии.
func (s *Service) appendHistory(playerID, gameID string, before, after int, result string, createdAt time.Time) {
	s.history[playerID] = append(s.history[playerID], HistoryEntry{
		ID:           platform.NewID(),
		PlayerID:     playerID,
		GameID:       gameID,
		RatingBefore: before,
		RatingAfter:  after,
		RatingDelta:  after - before,
		Result:       result,
		CreatedAt:    createdAt,
	})
}

// expectedScore — стандартная часть Elo-формулы: ожидаемый результат игрока.
func expectedScore(playerRating, opponentRating int) float64 {
	return 1 / (1 + math.Pow(10, float64(opponentRating-playerRating)/400))
}

// recalculateRating считает новый рейтинг по формуле Elo.
func recalculateRating(oldRating, kFactor int, actualScore, expected float64) int {
	return int(math.Round(float64(oldRating) + float64(kFactor)*(actualScore-expected)))
}

// actualScores переводит результат партии в числовую форму для Elo.
func actualScores(result string) (float64, float64) {
	switch result {
	case "white_win":
		return 1, 0
	case "black_win":
		return 0, 1
	default:
		return 0.5, 0.5
	}
}

// playerResult нужен для истории рейтинга конкретного игрока.
func playerResult(color, gameResult string) string {
	switch gameResult {
	case "draw":
		return "draw"
	case "white_win":
		if color == "white" {
			return "win"
		}
		return "loss"
	default:
		if color == "black" {
			return "win"
		}
		return "loss"
	}
}

// cloneProfile возвращает копию профиля, чтобы внешний код не менял внутреннее состояние map.
func cloneProfile(source *Profile) *Profile {
	if source == nil {
		return nil
	}

	clone := *source
	return &clone
}
