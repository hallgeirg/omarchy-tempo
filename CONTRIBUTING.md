# Contributing

Tempo is an early Go terminal app with a small QML bar widget. Bug reports and focused pull requests are welcome.

## Development

Use the Go version in `src/go.mod`. From the repository root:

```bash
make build
make test
make vet
make race
./bin/tempo --demo
./bin/tempo status
```

Tests use a local HTTP test server and temporary data directories; no live token is needed. Network access is needed to download Go dependencies. Format Go changes with `gofmt`.

## Layout

- `src/core.go`: Toggl HTTP client, shared cache, timer operations, reports, CSV.
- `src/main.go`: Bubble Tea dashboard, forms, demo data, CLI.
- `src/core_test.go`: API mock, file permissions, date boundaries, UI regression tests.
- `Widget.qml`: Omarchy bar status and click-to-open integration.
- `bin/tempo`: executable wrapper.
- `bin/tempo-window`: Omarchy open/focus launcher.

## Design expectations

Keep credentials out of logs, snapshots, command arguments, and fixtures. Use mocked HTTP tests for account-changing behavior. Avoid automatic write retries: a network error can occur after Toggl has accepted an operation. Preserve local cache sharing and rate-limit awareness. Demo mode must not alter a real account.

Describe the problem, resulting behavior, and relevant verification in a pull request. Avoid including private screenshots or account data. For releases, rebuild the bundled Linux x86-64 binary from the committed source and update the manifest version and changelog.
