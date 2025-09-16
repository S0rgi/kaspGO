package backoff

import (
	"math/rand"
	"time"
)

// Policy задаёт задержку перед ретраем по номеру попытки (начиная с 1).
type Policy interface {
	Delay(attempt int) time.Duration
}

// ExponentialJitter — экспоненциальная задержка с джиттером и верхней границей.
type ExponentialJitter struct {
	Base   time.Duration // базовая задержка (например, 50ms)
	Max    time.Duration // верхняя граница (например, 5s)
	Jitter time.Duration // до +/-Jitter добавляется случайно
}

// Delay вычисляет задержку для попытки attempt с экспоненциальным ростом и джиттером.
func (e ExponentialJitter) Delay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	d := e.Base
	for i := 1; i < attempt; i++ {
		d *= 2
		if d > e.Max {
			d = e.Max
			break
		}
	}
	// джиттер +/- Jitter/2
	if e.Jitter > 0 {
		delta := time.Duration(rand.Int63n(int64(e.Jitter))) - e.Jitter/2
		d += delta
		if d < 0 {
			d = 0
		}
	}
	return d
}
