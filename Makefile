.PHONY: all
all: test tool

.PHONY: tool
tool:
	go build -o build/evercoregen ./cmd/tool/

# Code generation
.PHONY: sqlc-gen
sqlc-gen:
	@echo "Generating SQL code..."
	go tool sqlc -f ./sqlite3.yaml generate
	go tool sqlc -f ./postgres.yaml generate

# Development
.PHONY: scratch
scratch: 
	go run scratch/main.go

# Testing
.PHONY: test
test: 
	go test -count=1 -race -coverprofile=coverage.out ./...

.PHONY: test-cover
test-cover: test
	go tool cover -html=coverage.out

# Integration testing
.PHONY: integration-test-sqlite 
integration-test-sqlite:
	go test -count=1 -race -tags=integration ./evercoresqlite

.PHONY: integration-test-postgres 
integration-test-postgres:
	go test -count=1 -race -tags=integration ./evercorepostgres

.PHONY: integration-test
integration-test: integration-test-sqlite integration-test-postgres

# Cleanup
.PHONY: clean
clean:
	rm -f coverage.out

