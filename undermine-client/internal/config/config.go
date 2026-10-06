package config

import (
	"bufio"
	"os"
	"strings"
)

// LoadEnv читает файл .env и возвращает переменные окружения
func LoadEnv(filename string) (map[string]string, error) {
	env := make(map[string]string)

	file, err := os.Open(filename)
	if err != nil {
		return env, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Пропускаем пустые строки и комментарии
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Разделяем по первому =
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Убираем кавычки если есть
		value = strings.Trim(value, `"'`)

		env[key] = value
	}

	if err := scanner.Err(); err != nil {
		return env, err
	}

	return env, nil
}

// GetEnv возвращает значение переменной окружения или значение по умолчанию
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// LoadServerURL загружает адрес сервера из .env файла
func LoadServerURL() string {
	// Сначала проверяем переменную окружения
	if url := os.Getenv("SERVER_URL"); url != "" {
		return url
	}

	// Затем читаем из .env файла
	env, err := LoadEnv(".env")
	if err != nil {
		return ""
	}

	return env["SERVER_URL"]
}
