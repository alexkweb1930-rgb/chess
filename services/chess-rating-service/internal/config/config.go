package config

import (
	"errors"
	"fmt"
	"path/filepath"

	"example.com/chess-platform/internal/platform"
)

type Config struct {
	ServiceName            string `json:"serviceName"`
	Version                string `json:"version"`
	Env                    string `json:"env"`
	Port                   string `json:"port"`
	LogLevel               string `json:"logLevel"`
	ReadTimeoutSeconds     int    `json:"readTimeoutSeconds"`
	WriteTimeoutSeconds    int    `json:"writeTimeoutSeconds"`
	ShutdownTimeoutSeconds int    `json:"shutdownTimeoutSeconds"`
	InitialRating          int    `json:"initialRating"`
	KFactor                int    `json:"kFactor"`
}

// Load собирает конфиг рейтинг-сервиса из файла и env.
func Load() (Config, error) {
	if err := platform.LoadDotEnvIfExists(".env"); err != nil {
		return Config{}, err
	}

	env := platform.GetEnv("APP_ENV", "dev")
	path := platform.GetEnv("CHESS_RATING_CONFIG", filepath.Join("config", "rating", env+".yaml"))

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

// applyDefaults подставляет стартовые значения для локальной разработки.
func applyDefaults(cfg *Config, env string) {
	if cfg.ServiceName == "" {
		cfg.ServiceName = "chess-rating-service"
	}
	if cfg.Version == "" {
		cfg.Version = "0.1.0"
	}
	if cfg.Env == "" {
		cfg.Env = env
	}
	if cfg.Port == "" {
		cfg.Port = "8081"
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
	if cfg.InitialRating == 0 {
		cfg.InitialRating = 1200
	}
	if cfg.KFactor == 0 {
		cfg.KFactor = 32
	}
}

// applyEnvOverrides позволяет не менять файлы между dev/test/prod.
func applyEnvOverrides(cfg *Config) {
	cfg.Port = platform.GetEnv("RATING_HTTP_PORT", cfg.Port)
	cfg.LogLevel = platform.GetEnv("LOG_LEVEL", cfg.LogLevel)
	cfg.Version = platform.GetEnv("APP_VERSION", cfg.Version)
	cfg.ReadTimeoutSeconds = platform.GetEnvInt("HTTP_READ_TIMEOUT_SECONDS", cfg.ReadTimeoutSeconds)
	cfg.WriteTimeoutSeconds = platform.GetEnvInt("HTTP_WRITE_TIMEOUT_SECONDS", cfg.WriteTimeoutSeconds)
	cfg.ShutdownTimeoutSeconds = platform.GetEnvInt("HTTP_SHUTDOWN_TIMEOUT_SECONDS", cfg.ShutdownTimeoutSeconds)
	cfg.InitialRating = platform.GetEnvInt("RATING_INITIAL_RATING", cfg.InitialRating)
	cfg.KFactor = platform.GetEnvInt("RATING_K_FACTOR", cfg.KFactor)
}

// validate останавливает старт, если обязательные настройки невалидны.
func validate(cfg Config) error {
	var errs []error

	if cfg.ServiceName == "" {
		errs = append(errs, errors.New("serviceName is required"))
	}
	if cfg.Port == "" {
		errs = append(errs, errors.New("port is required"))
	}
	if cfg.InitialRating <= 0 {
		errs = append(errs, errors.New("initialRating must be positive"))
	}
	if cfg.KFactor <= 0 {
		errs = append(errs, errors.New("kFactor must be positive"))
	}

	if len(errs) == 0 {
		return nil
	}

	return errors.Join(errs...)
}
