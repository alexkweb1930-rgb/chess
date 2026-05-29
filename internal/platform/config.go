package platform

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

var loadDotEnvOnce sync.Once

// LoadConfigFile читает конфиг из файла и декодирует его в переданную структуру.
// Сейчас конфиги хранятся в JSON-совместимом YAML, чтобы не тянуть внешние зависимости.
func LoadConfigFile(path string, target any) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config %q: %w", path, err)
	}

	if err := json.Unmarshal(content, target); err != nil {
		return fmt.Errorf("decode config %q: %w", path, err)
	}

	return nil
}

// LoadDotEnvIfExists один раз за процесс подхватывает корневой .env, если он есть.
// Это удобно для локальной разработки: можно не выставлять APP_ENV руками при каждом запуске.
// Уже заданные переменные окружения не перезаписываются, чтобы shell и Docker имели больший приоритет.
func LoadDotEnvIfExists(path string) error {
	var loadErr error

	loadDotEnvOnce.Do(func() {
		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return
			}

			loadErr = fmt.Errorf("read dotenv %q: %w", path, err)
			return
		}

		lines := strings.Split(string(content), "\n")
		for _, rawLine := range lines {
			line := strings.TrimSpace(rawLine)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}

			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			value = strings.Trim(value, `"'`)
			if key == "" {
				continue
			}

			if strings.TrimSpace(os.Getenv(key)) != "" {
				continue
			}

			if err := os.Setenv(key, value); err != nil {
				loadErr = fmt.Errorf("set dotenv var %q: %w", key, err)
				return
			}
		}
	})

	return loadErr
}

// GetEnv возвращает значение переменной окружения или fallback, если она не задана.
func GetEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

// GetEnvInt делает то же самое для целых чисел.
// Если значение не задано или не парсится, остается fallback.
func GetEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}
