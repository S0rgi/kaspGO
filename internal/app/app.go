package app

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"kaspContainers/internal/backoff"
	"kaspContainers/internal/config"
	"kaspContainers/internal/jobqueue"
	"kaspContainers/internal/processing"
)

// App инкапсулирует конфигурацию сервиса, очередь задач,
// процессор обработки и политику бэкоффа, а также управляет HTTP-сервером
// и жизненным циклом воркеров.
type App struct {
	cfg  config.Config
	q    *jobqueue.Queue
	proc processing.Processor
	bo   backoff.Policy
}

// New создаёт и возвращает новый экземпляр приложения.
func New(cfg config.Config, q *jobqueue.Queue, proc processing.Processor, bo backoff.Policy) *App {
	return &App{cfg: cfg, q: q, proc: proc, bo: bo}
}

// Run запускает HTTP-сервер, воркеры и ожидает завершения по ctx.
func (a *App) Run(ctx context.Context, addr string) error {
	acceptingMu := &sync.Mutex{}
	accepting := true
	mux := a.buildMux(acceptingMu, &accepting)
	srv := &http.Server{Addr: addr, Handler: mux}

	var wgWorkers sync.WaitGroup
	a.startWorkers(&wgWorkers)
	a.startServer(srv)

	<-ctx.Done()
	a.gracefulStop(srv, acceptingMu, &accepting, &wgWorkers)
	return nil
}

// buildMux настраивает маршруты HTTP: swagger, docs, healthz и enqueue.
func (a *App) buildMux(acceptingMu *sync.Mutex, accepting *bool) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/swagger/", http.StripPrefix("/swagger/", http.FileServer(http.Dir("docs/swagger"))))
	mux.Handle("/docs/", http.StripPrefix("/docs/", http.FileServer(http.Dir("docs"))))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/enqueue", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.ContentLength > 1<<20 {
			http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
			return
		}
		acceptingMu.Lock()
		if !*accepting {
			acceptingMu.Unlock()
			http.Error(w, "shutting down", http.StatusServiceUnavailable)
			return
		}
		acceptingMu.Unlock()

		var req struct {
			ID         string `json:"id"`
			Payload    string `json:"payload"`
			MaxRetries int    `json:"max_retries"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.ID == "" {
			http.Error(w, "id required", http.StatusBadRequest)
			return
		}
		if len(req.ID) > 128 {
			http.Error(w, "id too long", http.StatusBadRequest)
			return
		}
		if len(req.Payload) > 1<<20 {
			http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
			return
		}
		if req.MaxRetries < 0 || req.MaxRetries > 10 {
			http.Error(w, "max_retries must be between 0 and 10", http.StatusBadRequest)
			return
		}
		job := jobqueue.Job{ID: req.ID, Payload: req.Payload, MaxRetries: req.MaxRetries}
		if err := a.q.Enqueue(job); err != nil {
			if err == jobqueue.ErrClosed {
				log.Printf("enqueue rejected: closed id=%s", req.ID)
				http.Error(w, "queue closed", http.StatusServiceUnavailable)
				return
			}
			if err == jobqueue.ErrFull {
				log.Printf("enqueue rejected: full id=%s", req.ID)
				http.Error(w, "queue full", http.StatusTooManyRequests)
				return
			}
			log.Printf("enqueue error id=%s: %v", req.ID, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		log.Printf("enqueued id=%s max_retries=%d", req.ID, req.MaxRetries)
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
	})
	return mux
}

// startWorkers запускает пул воркеров, которые читают задания из очереди
// и обрабатывают их с ретраями по политике бэкоффа.
func (a *App) startWorkers(wg *sync.WaitGroup) {
	wg.Add(a.cfg.Workers)
	for i := 0; i < a.cfg.Workers; i++ {
		go func() {
			defer wg.Done()
			for {
				job, ok := a.q.Next()
				if !ok {
					return
				}
				start := time.Now()
				a.q.UpdatesStateRunning(job.ID)
				log.Printf("start id=%s", job.ID)
				maxAttempts := job.MaxRetries + 1
				for attempt := 1; attempt <= maxAttempts; attempt++ {
					ok, _ := a.proc.Process(job.ID, job.Payload)
					if ok {
						a.q.UpdatesStateDone(job.ID)
						log.Printf("done id=%s attempts=%d dur=%s", job.ID, attempt, time.Since(start))
						break
					}
					if attempt == maxAttempts {
						a.q.UpdatesStateFailed(job.ID)
						log.Printf("failed id=%s attempts=%d dur=%s", job.ID, attempt, time.Since(start))
						break
					}
					time.Sleep(a.bo.Delay(attempt))
				}
			}
		}()
	}
}

// startServer запускает HTTP-сервер в отдельной горутине.
func (a *App) startServer(srv *http.Server) {
	go func() {
		log.Printf("listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()
}

// gracefulStop прекращает приём новых задач, закрывает очередь,
// дожидается завершения воркеров и останавливает HTTP-сервер.
func (a *App) gracefulStop(srv *http.Server, acceptingMu *sync.Mutex, accepting *bool, wg *sync.WaitGroup) {
	acceptingMu.Lock()
	*accepting = false
	acceptingMu.Unlock()
	a.q.Close()
	wg.Wait()
	_ = srv.Shutdown(context.Background())
}
