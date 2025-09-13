package worker_pool

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSubmitExecutesTask(t *testing.T) {
	wp := NewWorkerPool(2)
	defer wp.StopWait()

	var executed int32
	var wg sync.WaitGroup
	wg.Add(1)

	wp.Submit(func() {
		atomic.AddInt32(&executed, 1)
		wg.Done()
	})

	// Ждём пока точно выполнится
	wg.Wait()

	if atomic.LoadInt32(&executed) != 1 {
		t.Errorf("expected task to be executed, got %d", executed)
	}
}
func TestSubmitWaitBlocksUntilDone(t *testing.T) {
	wp := NewWorkerPool(1)
	defer wp.StopWait()

	var executed int32

	wp.SubmitWait(func() {
		atomic.AddInt32(&executed, 1)
	})

	// если SubmitWait отработал, то задача уже точно завершена
	if atomic.LoadInt32(&executed) != 1 {
		t.Errorf("expected task to be executed, got %d", executed)
	}
}

func TestStopDoesNotRunQueuedTasks(t *testing.T) {
	wp := NewWorkerPool(1)

	var executed int32

	// отправим 2 задачи
	wp.Submit(func() {
		time.Sleep(100 * time.Millisecond) // имитируем долгую задачу
		atomic.AddInt32(&executed, 1)
	})
	wp.Submit(func() {
		atomic.AddInt32(&executed, 1)
	})

	// остановим пул
	wp.Stop()

	// ждём немного
	time.Sleep(200 * time.Millisecond)

	count := atomic.LoadInt32(&executed)
	if count < 1 || count > 2 {
		t.Errorf("expected 1 or 2 tasks to run, got %d", count)
	}
}
func TestMultipleTasks(t *testing.T) {
	wp := NewWorkerPool(4)
	defer wp.StopWait()

	var executed int32
	const tasksCount = 10
	var wg sync.WaitGroup
	wg.Add(tasksCount)

	for i := 0; i < tasksCount; i++ {
		wp.Submit(func() {
			atomic.AddInt32(&executed, 1)
			wg.Done()
		})
	}

	wg.Wait()

	if atomic.LoadInt32(&executed) != tasksCount {
		t.Errorf("expected %d tasks executed, got %d", tasksCount, executed)
	}
}
func TestStop(t *testing.T) {
	wp := NewWorkerPool(2)

	// сразу останавливаем
	wp.Stop()

	// проверяем, что отправка задач не зависает
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { recover() }() // так как канал закрыт, Submit вызовет панику
		wp.Submit(func() {})         // это должно паникнуть
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Submit blocked after Stop")
	}
}
func TestStopWaitWaitsForTasks(t *testing.T) {
	wp := NewWorkerPool(1)

	var executed int32
	wp.Submit(func() {
		time.Sleep(50 * time.Millisecond)
		atomic.AddInt32(&executed, 1)
	})

	start := time.Now()
	wp.StopWait()
	elapsed := time.Since(start)

	if atomic.LoadInt32(&executed) != 1 {
		t.Error("expected task to complete before StopWait returns")
	}
	if elapsed < 50*time.Millisecond {
		t.Error("StopWait returned too early")
	}
}
