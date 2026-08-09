// Package store implements the ledger.Store interface against SQLite.
//
// It owns the SQLite persistence layer via modernc.org/sqlite
// (pure Go, no CGO): file-backed for the server, in-memory for tests. The
// interface it satisfies is declared in internal/ledger, not here — the
// dependency points inward.
//
// Schema note: no uniqueness constraint beyond the primary key. This is
// deliberate and load-bearing; see .docs/decisions/.
package store
