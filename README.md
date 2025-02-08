# Evercore - event store library for Go

## Running tests

### Non-integration tests

```bash
make test
```

### Integation tests (all databases)

```bash
make integraion-test
```

### Sqlite integration tests

```bash
make integraion-test-sqlite
```

### Postgres integration tests

```bash
export PG_TEST_RUNNER_CONNECTION=<connection string to local postgres instance>
make integraion-test-postgres
```
