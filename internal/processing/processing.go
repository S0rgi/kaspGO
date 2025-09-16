package processing

import (
	"math/rand"
	"time"
)

// Processor инкапсулирует бизнес-логику обработки задания.
// Process выполняет задание и возвращает успех и длительность выполнения.
type Processor interface {
	Process(jobID string, payload string) (ok bool, attemptDuration time.Duration)
}

// RandomProcessor — пример реализации: случайная длительность и вероятность ошибки.
type RandomProcessor struct {
	ErrorRate int // 0..100
}

// Process имитирует обработку задания: случайная длительность 100-500мс,
// случайный успех/неуспех по ErrorRate.
func (p RandomProcessor) Process(jobID string, payload string) (bool, time.Duration) {
	sleepMs := 100 + rand.Intn(401) // 100..500ms
	d := time.Duration(sleepMs) * time.Millisecond
	time.Sleep(d)
	ok := rand.Intn(100) >= p.ErrorRate
	return ok, d
}
