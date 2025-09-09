# Repository Guidelines

This guide helps contributors work effectively in the Evercore repository.

## Project Structure & Module Organization

- `base/`: Core event-sourcing primitives (aggregates, events, store).
- `evercoresqlite/`, `evercorepostgres/`: Storage engines and tests.
- `evercoreuri/`: URI helpers and parsing.
- `cmd/evercoregen/`: Code generation tool binary source.
- `integrationtests/`, `enginetests/`: Integration and engine-focused tests.
- `examples/`: Minimal usage samples.
- `build/`, `tmp/`: Generated artifacts and coverage outputs.

## Build, Test, and Development Commands

- Build tool: `make tool` — builds `build/evercoregen`.
- Unit tests: `make test` — runs `./...` with `-race` and coverage.
- Integration tests: `make integration-test` — all backends with `-tags=integration`.
  - Backend-specific: `make integration-test-sqlite`, `make integration-test-postgres`.
- Coverage reports: `make test-cover`, `make integration-test-cover`.
- SQL codegen: `make sqlc-gen` — generates code from `.sql` via `sqlc` configs.
- Run scratch: `make scratch` — executes `scratch/main.go` for local experiments.
- Targeted test: `go test -run TestName ./path/to/pkg`.

## Coding Style & Naming Conventions

- Formatting: `gofmt` standard; organize imports (stdlib, third-party, local).
- Naming: PascalCase exported, camelCase unexported; acronyms all caps (ID, SQL).
- Errors: handle immediately; prefer `errors.Is/As`; wrap with context.
- Types: prefer explicit structs/interfaces; avoid globals.
- SQL: keep queries in `.sql`; regenerate via `make sqlc-gen`.

## Testing Guidelines

- Keep tests in the same package with `_test.go`; prefer table-driven tests.
- Use `-race`; split unit vs integration via `-tags=integration`.
- Generate coverage locally with `make test-cover` or integration profile.
- Backend tests may require env vars (e.g., `PG_TEST_RUNNER_CONNECTION=postgres://...`).

## Commit & Pull Request Guidelines

- Commits: follow Conventional Commits (feat:, fix:, chore:, refactor:, test:, docs:). Include a clear body.
- PRs: concise title, description, linked issues, test plan, and any schema/codegen changes noted.
- CI hygiene: run `make test` and relevant `make integration-test-*` targets before submitting.
