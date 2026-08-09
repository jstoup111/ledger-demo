package clock

import (
	"testing"
	"time"
)

func TestClocksProvideTime(t *testing.T) {
	fixed := time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
	clock := FixedClock{T: fixed}

	if got := clock.Now(); got != fixed || clock.Now() != fixed {
		t.Fatalf("FixedClock.Now() = %v; want %v on successive calls", got, fixed)
	}

	var system SystemClock
	if system.Now().IsZero() {
		t.Fatal("SystemClock.Now() returned the zero time")
	}
}
