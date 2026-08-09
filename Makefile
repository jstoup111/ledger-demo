.PHONY: dev seed test reset fmt vet check

PORT ?= 8080
LEDGER_DB_PATH ?= ./ledger.db

# Run the server on a fixed port.
dev:
	PORT=$(PORT) LEDGER_DB_PATH=$(LEDGER_DB_PATH) go run ./cmd/server serve

# Drop the DB and load deterministic seed data.
seed:
	rm -f $(LEDGER_DB_PATH) $(LEDGER_DB_PATH)-shm $(LEDGER_DB_PATH)-wal
	LEDGER_DB_PATH=$(LEDGER_DB_PATH) go run ./cmd/server seed

# Run the suite.
test:
	go test ./...

# Single command restoring pristine demo state.
reset: seed
	@echo "reset: pristine demo state restored (db=$(LEDGER_DB_PATH))"

fmt:
	gofmt -l -w .

vet:
	go vet ./...

# What the pre-PR lint gate runs.
check:
	@test -z "$$(gofmt -l .)" || { echo "gofmt needed:"; gofmt -l .; exit 1; }
	go vet ./...
