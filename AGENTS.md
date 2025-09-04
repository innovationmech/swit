# Repository Guidelines

## Project Structure & Modules
- `cmd/`: Service/CLI entrypoints (`swit-serve`, `swit-auth`, `switctl`).
- `pkg/`: Reusable framework packages (server, transport, middleware, discovery).
- `internal/`: Reference services and internal-only code.
- `api/proto` → sources; `api/gen` → generated code; keep gen code committed.
- `examples/`: Minimal HTTP/gRPC and full-featured samples.
- `docs/`: Generated API docs and guides; `_output/`: build artifacts.
- `scripts/`: Makefile includes and tooling wrappers.

## Build, Test, Develop
- `make setup-dev`: Install dev tools (buf, swag, quality checks).
- `make build` / `make build-dev`: Build all services (dev fast path skips full checks).
- `make test`: Run all tests across `internal` and `pkg` (generates deps).
- `make test-coverage`: Create `coverage.out` and `coverage.html`.
- `make proto` / `make swagger`: Generate protobuf stubs and OpenAPI docs.
- `make quality`: Tidy, format, vet, lint; `make quality-dev` for a faster pass.
- Docker: `make docker`; run binaries from `./bin/` or images as needed.

## Coding Style & Naming
- Language: Go 1.23+. Format with `gofmt`; organize imports with `goimports`.
- Packages: lower-case, no underscores (e.g., `transport`, `middleware`).
- Files: `snake_case.go`; tests in `*_test.go` beside code.
- Exports: Use Go idioms (`CamelCase`, clear godoc comments on exported items).
- Protos: keep sources in `api/proto`, generated code in `api/gen` (do not edit).

## Testing Guidelines
- Framework: standard `go test`; table-driven tests preferred.
- Scope helpers: `make test-advanced TYPE=race PACKAGE=internal` (also: `unit`, `bench`, `short`).
- Coverage: keep meaningful unit coverage; verify via `make test-coverage` and review `coverage.html`.

## Commit & PR Guidelines
- Style: Prefer Conventional Commits (`feat:`, `fix:`, `docs:`). Keep subject imperative; add scope when useful.
- PRs must include: clear summary, linked issues (e.g., `Closes #123`), test updates, and docs when APIs change.
- Pre-flight: `make quality && make test` must pass; include screenshots/logs for behavior changes.

## Security & Configuration
- Do not commit secrets. Use local env/config files; check `swit.yaml`, `switauth.yaml`, and service-specific YAML in repo as templates.
- Validate inputs and timeouts when adding middleware/handlers; run `make quality-advanced OPERATION=security` for static checks if installed.

## Architecture Notes
- HTTP/gRPC coordination and DI live under `pkg/`; examples and `internal/*` show recommended integration patterns. Mirror these when adding new services under `cmd/`.

