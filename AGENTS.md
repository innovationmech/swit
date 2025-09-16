# Repository Guidelines

## Project Structure & Module Organization
The repository centers around Go services and shared messaging libraries. Entry points for binaries live under `cmd/` (for example `cmd/swit-serve`). Reusable packages that power messaging, transport, and middleware sit in `pkg/`, while reference implementations and service-specific logic reside in `internal/`. API definitions are split between `api/proto/` (source `.proto` files) and `api/gen/` (generated code that must stay committed). Supporting documentation and build artifacts can be found in `docs/` and `docs/_output/`, with tooling helpers collected in `scripts/`.

## Build, Test, and Development Commands
Run `make setup-dev` once to install buf, swag, and the quality toolchain. `make build` produces all binaries with full checks; use `make build-dev` for a faster developer iteration. Execute `make test` (or `go test ./pkg/messaging/rabbitmq -v` when tightening scope) to run unit tests, and `make test-coverage` to generate `coverage.out` and `coverage.html`. Format, lint, and vet the codebase with `make quality`; `make quality-dev` offers a lighter pass. Regenerate protobufs via `make proto`, OpenAPI specs via `make swagger`, and container images with `make docker`.

## Coding Style & Naming Conventions
Code targets Go 1.23+ with `gofmt` and `goimports` enforcing formatting and import order. Package names remain lower-case without underscores; Go filenames prefer `snake_case.go` with tests co-located as `*_test.go`. Stick to idiomatic CamelCase for exported identifiers, provide godoc comments on exported types and functions, and keep configuration structures in YAML/JSON aligned with existing tags.

## Testing Guidelines
Tests rely on the standard library and favor table-driven patterns. When feasible, scope runs using `go test ./pkg/...` or issue-specific subsets. Use `make test-coverage` to inspect coverage artifacts and keep meaningful assertions. Name tests with the feature under test (e.g., `TestRabbitPublisherPublishWithConfirm`) and ensure mocks in `pkg/messaging/rabbitmq/mocks_test.go` stay consistent with the AMQP abstraction.

## Commit & Pull Request Guidelines
Follow Conventional Commit prefixes (`feat:`, `fix:`, `docs:`, etc.) with imperative subjects and optional scopes (`feat(rabbitmq): ...`). Before opening a PR, run `make quality` and `make test`; capture command output or screenshots for behavioral changes. PR descriptions should summarize the change, link related issues using keywords like `Closes #301`, and note any configuration or migration steps. Allow maintainers to edit the branch when possible and keep the change surface focused.

## Security & Configuration Tips
Never commit secrets—use the sample configs (`swit.yaml`, `switauth.yaml`, service-specific YAML) as templates. Validate broker endpoints, TLS options, and timeouts when updating messaging adapters, and lean on `make quality-advanced OPERATION=security` if the extended checks are available.
