// Package clock provides injectable time.
//
// This package is deliberately empty; it is built during the demo alongside the
// domain that depends on it.
//
// When it is built, it owns:
//
//	type Clock interface { Now() time.Time }
//
// plus SystemClock (the only place in the entire codebase permitted to call
// time.Now) and FixedClock (used by every test, so the suite is fully
// deterministic).
//
// Time is injected, never read directly.
package clock
