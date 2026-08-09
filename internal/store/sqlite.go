package store

// The pure-Go SQLite driver (no CGO), registered with database/sql under the
// name "sqlite".
//
// This blank import is here before any code uses it on purpose: it pins
// modernc.org/sqlite in go.mod and go.sum so the module is already resolved and
// cached. The demo must build and run with no network, and discovering a missing
// dependency on stage is not recoverable.
import _ "modernc.org/sqlite"
