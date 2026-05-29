package config

import (
	"errors"
	"fmt"
	"path/filepath"

	"example.com/chess-platform/internal/platform"
)

type Config struct {
	ServiceName                 string `json:"serviceName"`
	Version                     string `json:"version"`
	Env                         string `json:"env"`
	Port                        string `json:"port"`
	LogLevel                    string `json:"logLevel"`
	ReadTimeoutSeconds          int    `json:"readTimeoutSeconds"`
	WriteTimeoutSeconds         int    `json:"writeTimeoutSeconds"`
	ShutdownTimeoutSeconds      int    `json:"shutdownTimeoutSeconds"`
	RatingServiceBaseURL        string `json:"ratingServiceBaseURL"`
	RatingServiceTimeoutSeconds int    `json:"ratingServiceTimeoutSeconds"`
}

// Load собирает конфиг сервиса партий из файла и переменных окружения.
// Приоритет такой: defaults -> файл -> env overrides.
func Load() (Config, error) {
	if err := platform.LoadDotEnvIfExists(".env"); err != nil {
		return Config{}, err
	}

	env := platform.GetEnv("APP_ENV", "dev")
	path := platform.GetEnv("CHESS_GAME_CONFIG", filepath.Join("config", "game", env+".yaml"))

	cfg := Config{}
	if err := platform.LoadConfigFile(path, &cfg); err != nil {
		return Config{}, err
	}

	applyDefaults(&cfg, env)
	applyEnvOverrides(&cfg)

	if err := validate(cfg); err != nil {
		return Config{}, fmt.Errorf("validate config %q: %w", path, err)
	}

	return cfg, nil
}

// applyDefaults подставляет безопасные значения по умолчанию,
// чтобы сервис можно было быстро поднять локально.
func applyDefaults(cfg *Config, env string) {
	if cfg.ServiceName == "" {
		cfg.ServiceName = "chess-game-service"
	}
	if cfg.Version == "" {
		cfg.Version = "0.1.0"
	}
	if cfg.Env == "" {
		cfg.Env = env
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	if cfg.ReadTimeoutSeconds == 0 {
		cfg.ReadTimeoutSeconds = 5
	}
	if cfg.WriteTimeoutSeconds == 0 {
		cfg.WriteTimeoutSeconds = 10
	}
	if cfg.ShutdownTimeoutSeconds == 0 {
		cfg.ShutdownTimeoutSeconds = 10
	}
	if cfg.RatingServiceBaseURL == "" {
		cfg.RatingServiceBaseURL = "http://localhost:8081"
	}
	if cfg.RatingServiceTimeoutSeconds == 0 {
		cfg.RatingServiceTimeoutSeconds = 3
	}
}

// applyEnvOverrides нужен для запуска в разных окружениях без правки файлов.
func applyEnvOverrides(cfg *Config) {
	cfg.Port = platform.GetEnv("GAME_HTTP_PORT", cfg.Port)
	cfg.LogLevel = platform.GetEnv("LOG_LEVEL", cfg.LogLevel)
	cfg.Version = platform.GetEnv("APP_VERSION", cfg.Version)
	cfg.RatingServiceBaseURL = platform.GetEnv("RATING_SERVICE_BASE_URL", cfg.RatingServiceBaseURL)
	cfg.ReadTimeoutSeconds = platform.GetEnvInt("HTTP_READ_TIMEOUT_SECONDS", cfg.ReadTimeoutSeconds)
	cfg.WriteTimeoutSeconds = platform.GetEnvInt("HTTP_WRITE_TIMEOUT_SECONDS", cfg.WriteTimeoutSeconds)
	cfg.ShutdownTimeoutSeconds = platform.GetEnvInt("HTTP_SHUTDOWN_TIMEOUT_SECONDS", cfg.ShutdownTimeoutSeconds)
	cfg.RatingServiceTimeoutSeconds = platform.GetEnvInt("RATING_SERVICE_TIMEOUT_SECONDS", cfg.RatingServiceTimeoutSeconds)
}

// validate проверяет обязательные поля конфига до старта сервиса.
func validate(cfg Config) error {
	var errs []error

	if cfg.ServiceName == "" {
		errs = append(errs, errors.New("serviceName is required"))
	}
	if cfg.Port == "" {
		errs = append(errs, errors.New("port is required"))
	}
	if cfg.RatingServiceBaseURL == "" {
		errs = append(errs, errors.New("ratingServiceBaseURL is required"))
	}

	if len(errs) == 0 {
		return nil
	}

	return errors.Join(errs...)
}
