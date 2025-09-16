package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kaspContainers/internal/app"
	"kaspContainers/internal/backoff"
	"kaspContainers/internal/config"
	"kaspContainers/internal/jobqueue"
	"kaspContainers/internal/processing"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	cfg := config.Load()
	q := jobqueue.NewQueue(cfg.QueueSize)
	proc := processing.RandomProcessor{ErrorRate: cfg.ErrorRate}
	bo := backoff.ExponentialJitter{Base: 50 * time.Millisecond, Max: 5 * time.Second, Jitter: 50 * time.Millisecond}
	application := app.New(cfg, q, proc, bo)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-sigCh
		log.Println("shutting down...")
		cancel()
	}()

	_ = application.Run(ctx, ":8080")
}
