package config

import (
	"os"
	"strconv"
)

// Config содержит конфигурацию приложения, загружаемую из переменных окружения.
type Config struct {
	Workers   int
	QueueSize int
	ErrorRate int // 0..100, процент неуспеха обработки
}

// getenvInt читает переменную окружения как целое число или возвращает значение по умолчанию.
func getenvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// Load создаёт конфигурацию из переменных окружения.
func Load() Config {
	return Config{
		Workers:   getenvInt("WORKERS", 4),
		QueueSize: getenvInt("QUEUE_SIZE", 64),
		ErrorRate: getenvInt("ERROR_RATE", 20),
	}
}
