// Package ledger holds the domain: accounts, transactions, balances, and validation.
//
// This package is deliberately empty. The ledger domain is the subject of a live
// demo and is built during the presentation, not scaffolded ahead of it.
//
// When it is built, it owns:
//
//   - Account: ID, Name, and a balance DERIVED from transactions (never a stored
//     mutable field).
//   - Transaction: ID, AccountID, Amount (signed int64 minor units — cents),
//     Description, CreatedAt.
//   - Validation on post: account exists; amount is non-zero; description is
//     non-empty and <= 140 chars; a transaction that would take the balance below
//     zero is rejected.
//   - The Store interface — declared HERE and implemented in internal/store, so
//     this package has no knowledge of SQLite and unit-tests without a database.
//   - Sentinel errors, wrapped for errors.Is. Never bare errors.New at call
//     sites; never error strings compared by value.
//
// Money is int64 cents throughout. No floats anywhere, ever.
package ledger
