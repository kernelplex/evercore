.PHONY: test
test: 
	go test -count=1 ./... 

.PHONY: integration-test-sqlite 
integration-test-sqlite:
	go test -count=1 -tags=integration ./pkg/evercoresqlite

.PHONY: integration-test-postgres 
integration-test-postgres:
	go test -count=1 -tags=integration ./pkg/evercorepostgres

.PHONY: integration-test
integration-test: integration-test-sqlite integration-test-postgres

