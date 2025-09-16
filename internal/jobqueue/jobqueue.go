package jobqueue

import (
	"errors"
	"math/rand"
	"sync"
	"time"
)

// State представляет состояние задания в очереди.
type State string

const (
	StateQueued  State = "queued"
	StateRunning State = "running"
	StateDone    State = "done"
	StateFailed  State = "failed"
)

// Job представляет задание для обработки.
type Job struct {
	ID         string
	Payload    string
	MaxRetries int
}

type item struct {
	job Job
}

type Queue struct {
	ch        chan item
	mu        sync.Mutex
	idToState map[string]State
	closed    bool
}

// NewQueue создаёт новую очередь с заданным размером буфера.
func NewQueue(bufferSize int) *Queue {
	return &Queue{
		ch:        make(chan item, bufferSize),
		idToState: make(map[string]State),
	}
}

var ErrClosed = errors.New("queue closed")
var ErrFull = errors.New("queue full")

// Enqueue добавляет задание в очередь. Возвращает ошибку, если очередь закрыта или переполнена.
func (q *Queue) Enqueue(job Job) error {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return ErrClosed
	}
	q.idToState[job.ID] = StateQueued
	q.mu.Unlock()

	select {
	case q.ch <- item{job: job}:
		return nil
	default:
		q.mu.Lock()
		q.idToState[job.ID] = StateFailed
		q.mu.Unlock()
		return ErrFull
	}
}

// Close закрывает очередь для новых заданий.
func (q *Queue) Close() {
	q.mu.Lock()
	if !q.closed {
		q.closed = true
		close(q.ch)
	}
	q.mu.Unlock()
}

// Next блокирующе возвращает следующее задание из очереди.
// Возвращает ok=false, когда очередь закрыта и опустела.
func (q *Queue) Next() (Job, bool) {
	it, ok := <-q.ch
	if !ok {
		return Job{}, false
	}
	return it.job, true
}

// UpdatesStateRunning обновляет состояние задания на "выполняется".
func (q *Queue) UpdatesStateRunning(id string) {
	q.mu.Lock()
	q.idToState[id] = StateRunning
	q.mu.Unlock()
}

// UpdatesStateDone обновляет состояние задания на "завершено".
func (q *Queue) UpdatesStateDone(id string) {
	q.mu.Lock()
	q.idToState[id] = StateDone
	q.mu.Unlock()
}

// UpdatesStateFailed обновляет состояние задания на "неудачно".
func (q *Queue) UpdatesStateFailed(id string) {
	q.mu.Lock()
	q.idToState[id] = StateFailed
	q.mu.Unlock()
}

// StatesSnapshot возвращает снимок всех состояний заданий.
func (q *Queue) StatesSnapshot() map[string]State {
	q.mu.Lock()
	defer q.mu.Unlock()
	copy := make(map[string]State, len(q.idToState))
	for k, v := range q.idToState {
		copy[k] = v
	}
	return copy
}

// WorkerLoop обрабатывает задания из очереди до закрытия канала или завершения контекста done.
// simulateProcess имитирует обработку задачи и возвращает ok=true при успехе, иначе false.
// Устаревший метод, используется только в тестах.
func WorkerLoop(done <-chan struct{}, q *Queue, simulateProcess func(Job) bool) {
	for {
		select {
		case <-done:
			return
		case it, ok := <-q.ch:
			if !ok {
				return
			}
			job := it.job
			q.UpdatesStateRunning(job.ID)

			// ретраи с экспоненциальным бэкофом и джиттером
			var attempt int
			maxAttempts := job.MaxRetries + 1
			for {
				if simulateProcess(job) {
					q.UpdatesStateDone(job.ID)
					break
				}
				attempt++
				if attempt >= maxAttempts {
					q.UpdatesStateFailed(job.ID)
					break
				}
				// экспоненциальный бэкофф 50..100ms * 2^(attempt-1) с джиттером
				baseMs := 50 + rand.Intn(51) // 50..100
				backoff := time.Duration(baseMs) * time.Millisecond
				for i := 1; i < attempt; i++ {
					backoff *= 2
				}
				jitter := time.Duration(rand.Intn(50)) * time.Millisecond
				select {
				case <-done:
					return
				case <-time.After(backoff + jitter):
				}
			}
		}
	}
}
