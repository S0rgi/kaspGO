package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"kaspContainers/internal/backoff"
	"kaspContainers/internal/config"
	"kaspContainers/internal/jobqueue"
)

// dummyProc всегда успешно "обрабатывает" задачу без задержки
type dummyProc struct{}

func (dummyProc) Process(jobID string, payload string) (bool, time.Duration) {
	return true, 0
}

func newTestApp() *App {
	cfg := config.Config{Workers: 1, QueueSize: 8, ErrorRate: 0}
	q := jobqueue.NewQueue(cfg.QueueSize)
	bo := backoff.ExponentialJitter{Base: 1 * time.Millisecond, Max: 2 * time.Millisecond, Jitter: 0}
	return New(cfg, q, dummyProc{}, bo)
}

func TestHealthz(t *testing.T) {
	a := newTestApp()
	mux := a.buildMux(&sync.Mutex{}, boolPtr(true))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func boolPtr(b bool) *bool { return &b }

func TestEnqueueAcceptsAndProcesses(t *testing.T) {
	a := newTestApp()
	accepting := true
	mux := a.buildMux(&sync.Mutex{}, &accepting)
	var wg sync.WaitGroup
	a.startWorkers(&wg)

	body := map[string]any{"id": "t1", "payload": "p", "max_retries": 0}
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(body)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/enqueue", &buf)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}

	// дождёмся обработки
	time.Sleep(20 * time.Millisecond)
	st := a.q.StatesSnapshot()["t1"]
	if st != jobqueue.StateDone {
		t.Fatalf("expected done, got %v", st)
	}
	// корректно завершим воркеров
	a.q.Close()
	wg.Wait()
}

func TestEnqueueRejectsOnShutdown(t *testing.T) {
	a := newTestApp()
	accepting := false
	mux := a.buildMux(&sync.Mutex{}, &accepting)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/enqueue", bytes.NewBufferString(`{"id":"x"}`))
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
}

func TestRunAndShutdown(t *testing.T) {
	a := newTestApp()
	ctx, cancel := context.WithCancel(context.Background())
	srv := httptest.NewServer(a.buildMux(&sync.Mutex{}, boolPtr(true)))
	defer srv.Close()
	cancel()
	_ = a.Run(ctx, ":0")
}
