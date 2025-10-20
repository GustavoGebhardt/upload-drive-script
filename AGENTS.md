# Repository Guidelines

## Project Structure & Module Organization
- `cmd/main.go` wires the HTTP server and routing; keep new binaries under `cmd/<name>/`.
- `internal/config`, `internal/handlers`, and `internal/services` hold configuration loading, HTTP endpoints, and Drive-specific orchestration respectively. Add new cross-cutting packages here to keep external APIs clean.
- `pkg/logger` exposes reusable logging helpers. Prefer extending this package instead of embedding new loggers in handlers.
- Secrets such as `credentials.json` and the runtime `token.json` live at the repo root; never commit modified production tokens.

## Build, Test, and Development Commands
- `go mod tidy` ensures dependencies in `go.mod` stay in sync when packages change.
- `go build -o upload-drive-script ./cmd` produces the CLI/server binary used in production.
- `HTTP_LISTEN_ADDR=:3000 go run ./cmd/main.go` starts the local server; adjust the env var instead of editing code.
- `go test ./... -cover` executes the Go unit tests with a coverage report; run before opening a pull request.

## Coding Style & Naming Conventions
- Format Go code with `go fmt ./...` (the repo uses Go 1.25 defaults: tabs for indentation, trailing newline required).
- Follow idiomatic Go naming: packages lower_snake, exported types/functions in PascalCase only when needed outside the package.
- Keep handler names verbs (`Upload`, `UploadURL`) and service methods explicit (`UploadFromForm` style) to mirror HTTP semantics.
- Use `pkg/logger.Info/Error` for structured logging; add helpers there rather than scattering `log.Print*` calls.

## Testing Guidelines
- Place tests beside the code (`internal/handlers/drive_handler_test.go`), naming files `*_test.go` and functions `TestXxx`.
- Use `net/http/httptest` for router-level coverage and stub Drive interactions with interfaces to avoid real API calls.
- Aim to cover successful upload flows and error paths (invalid tokens, missing form file, download failures) before merging.

## Commit & Pull Request Guidelines
- Follow Conventional Commits (`feat: add resumable upload chunking`, `fix:` for bug patches) as seen in the Git history.
- Squash work-in-progress commits locally; each PR should read as a coherent change set with a clear title and checklist of tests run.
- PR descriptions must call out affected endpoints, configuration updates, and any manual QA steps (e.g., curl examples or Postman screenshots).

## Configuration & Security Tips
- Prefer environment variables over hardcoded values. Supported keys: `GOOGLE_CREDENTIALS_FILE`, `GOOGLE_TOKEN_FILE`, `GOOGLE_OAUTH_REDIRECT_URL`, `GOOGLE_OAUTH_STATE`, and `HTTP_LISTEN_ADDR`. All have safe defaults; document overrides in PRs.
- Keep OAuth credentials out of history; store environment-specific copies outside the repo and document local setup in the PR.
- When testing Drive integrations, use restricted scopes and revoke tokens in Google Cloud if a machine is decommissioned.
