package backoff

import (
	"testing"
	"time"
)

func TestExponentialJitterWithinBounds(t *testing.T) {
	bo := ExponentialJitter{Base: 50 * time.Millisecond, Max: 800 * time.Millisecond, Jitter: 50 * time.Millisecond}
	for attempt := 1; attempt <= 10; attempt++ {
		d := bo.Delay(attempt)
		if d < 0 {
			t.Fatalf("delay negative: %v", d)
		}
		// допускаем, что джиттер может слегка выйти за Max вверх, но не более чем на Jitter/2
		if d > bo.Max+bo.Jitter/2 {
			t.Fatalf("delay exceeds max + jitter/2: got %v (max %v, jitter %v)", d, bo.Max, bo.Jitter)
		}
	}
}

func TestExponentialJitterNoPanicOnZeroOrNegativeAttempt(t *testing.T) {
	bo := ExponentialJitter{Base: 10 * time.Millisecond, Max: 20 * time.Millisecond, Jitter: 5 * time.Millisecond}
	if d := bo.Delay(0); d < 0 || d > bo.Max+bo.Jitter/2 {
		t.Fatalf("unexpected delay for attempt=0: %v", d)
	}
	if d := bo.Delay(-1); d < 0 || d > bo.Max+bo.Jitter/2 {
		t.Fatalf("unexpected delay for attempt=-1: %v", d)
	}
}
