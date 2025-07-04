.PHONY: all
all: integration-test-setup test tool

.PHONY: tool
tool:
	go build -o build/evercoregen ./cmd/evercoregen/

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

.PHONY: integration-test-cover
integration-test-cover: tool integration-test-setup integration-test
	go tool cover -html=integration-test-profile.out -o ./tmp/integration-test-coverage.html




# Integration test setup
.PHONY: integation-test-setup
integration-test-setup: tool
	build/evercoregen -output-dir=integrationtests/generated/ -output-pkg=generated

# Integration testing
.PHONY: integration-test-sqlite 
integration-test-sqlite: integration-test-setup
	go test -count=1 -race -tags=integration ./evercoresqlite

.PHONY: integration-test-postgres 
integration-test-postgres: integration-test-setup
	go test -count=1 -race -tags=integration ./evercorepostgres

.PHONY: integration-test
integration-test: integration-test-setup 
	go test -count=1 -race -tags=integration -coverprofile=integration-test-profile.out ./...
# Cleanup
.PHONY: clean
clean:
	rm -f coverage.out

