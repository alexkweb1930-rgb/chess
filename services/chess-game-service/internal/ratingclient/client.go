package ratingclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type GameResultRequest struct {
	GameID        string    `json:"gameId"`
	WhitePlayerID string    `json:"whitePlayerId"`
	BlackPlayerID string    `json:"blackPlayerId"`
	Result        string    `json:"result"`
	FinishedAt    time.Time `json:"finishedAt"`
}

// New создает минимальный HTTP-клиент для внутреннего вызова рейтинг-сервиса.
func New(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// SendGameResult отправляет результат завершенной партии во второй микросервис.
// Здесь логика намеренно простая: один запрос, один ответ, без ретраев и очередей.
func (c *Client) SendGameResult(payload GameResultRequest) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal rating request: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, c.baseURL+"/internal/v1/rating/game-result", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build rating request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send rating request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("rating service returned status %d", response.StatusCode)
	}

	return nil
}
