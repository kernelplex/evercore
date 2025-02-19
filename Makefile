.PHONY: test
test: 
	go test -count=1 ./... 

.PHONY: integration-test-sqlite 
integration-test-sqlite:
	go test -count=1 -tags=integration ./evercoresqlite

.PHONY: integration-test-postgres 
integration-test-postgres:
	go test -count=1 -tags=integration ./evercorepostgres

.PHONY: integration-test
integration-test: integration-test-sqlite integration-test-postgres

