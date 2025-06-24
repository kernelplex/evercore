# Evercore Agent Guidelines

## Build & Test Commands

- Build: `make tool`
- Run all tests: `make test`
- Run integration tests: `make integration-test`
- Run specific test: `go test -run TestName ./path/to/pkg`
- Generate coverage report: `make test-cover`

## Code Style Conventions

1. **Formatting**: Use `gofmt` standard formatting
2. **Imports**: Group stdlib, third-party, local imports
3. **Naming**:
   - PascalCase for exported identifiers
   - camelCase for unexported
   - Acronyms all caps (e.g., SQL, ID)
4. **Error Handling**:
   - Handle errors immediately
   - Use `errors.Is`/`errors.As` for error inspection
   - Wrap errors with context
5. **Types**:
   - Use strong typing with structs/interfaces
   - Avoid global variables
6. **SQL**:
   - Generate SQL code with `make sqlc-gen`
   - Keep queries in .sql files

## Testing Guidelines

- Use `-race` flag for concurrency tests
- Prefer table-driven tests
- Separate unit and integration tests (`-tags=integration`)
- Keep tests in same package with `_test` suffix

## Commit Messages

- Use the [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/) format
- Include a body with a more detailed description of the changes
