package clock

import "time"

// Clock provides the current time.
type Clock interface {
	Now() time.Time
}

// SystemClock reads the current system time.
type SystemClock struct{}

// Now returns the current system time.
func (SystemClock) Now() time.Time {
	return time.Now()
}

// FixedClock always returns T.
type FixedClock struct {
	T time.Time
}

// Now returns the fixed time.
func (c FixedClock) Now() time.Time {
	return c.T
}
