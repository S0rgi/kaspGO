package jobqueue

import (
	"sync/atomic"
	"testing"
	"time"
)

// TestEnqueueAndStates проверяет, что задача попадает в очередь и меняет состояние на done
// после успешной обработки воркером.
func TestEnqueueAndStates(t *testing.T) {
	q := NewQueue(1)
	defer q.Close()

	done := make(chan struct{})
	defer close(done)

	processed := int32(0)
	go WorkerLoop(done, q, func(j Job) bool {
		atomic.AddInt32(&processed, 1)
		return true
	})

	if err := q.Enqueue(Job{ID: "a", Payload: "p", MaxRetries: 0}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	// подождём обработку
	time.Sleep(50 * time.Millisecond)

	st := q.StatesSnapshot()
	if st["a"] != StateDone {
		t.Fatalf("expected state done, got %v", st["a"])
	}
	if atomic.LoadInt32(&processed) != 1 {
		t.Fatalf("expected 1 process, got %d", processed)
	}
}

// TestQueueFull проверяет, что переполненная очередь возвращает ErrFull и помечает задачу failed.
func TestQueueFull(t *testing.T) {
	q := NewQueue(1)
	defer q.Close()

	// не запускаем воркера, чтобы канал остался занятым
	if err := q.Enqueue(Job{ID: "a"}); err != nil {
		t.Fatalf("first enqueue failed: %v", err)
	}
	if err := q.Enqueue(Job{ID: "b"}); err == nil {
		t.Fatalf("expected ErrFull, got nil")
	} else if err != ErrFull {
		t.Fatalf("expected ErrFull, got %v", err)
	}
	if q.StatesSnapshot()["b"] != StateFailed {
		t.Fatalf("expected b failed state on full queue")
	}
}

// TestRetriesWithBackoff проверяет, что при неудачах выполняются ретраи до успеха
// и что при исчерпании ретраев задача помечается failed.
func TestRetriesWithBackoff(t *testing.T) {
	// успешные ретраи
	{
		q := NewQueue(1)
		done := make(chan struct{})

		attempts := int32(0)
		go WorkerLoop(done, q, func(j Job) bool {
			c := atomic.AddInt32(&attempts, 1)
			if c < 3 {
				return false
			}
			return true
		})

		if err := q.Enqueue(Job{ID: "x", MaxRetries: 5}); err != nil {
			t.Fatalf("enqueue failed: %v", err)
		}

		time.Sleep(500 * time.Millisecond)
		if q.StatesSnapshot()["x"] != StateDone {
			t.Fatalf("expected x done, got %v", q.StatesSnapshot()["x"])
		}

		close(done)
		q.Close()
	}

	// исчерпание ретраев
	{
		q := NewQueue(1)
		done := make(chan struct{})

		go WorkerLoop(done, q, func(j Job) bool { return false })

		if err := q.Enqueue(Job{ID: "y", MaxRetries: 1}); err != nil {
			t.Fatalf("enqueue failed: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
		if q.StatesSnapshot()["y"] != StateFailed {
			t.Fatalf("expected y failed, got %v", q.StatesSnapshot()["y"])
		}

		close(done)
		q.Close()
	}
}
