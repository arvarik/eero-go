# Testing Strategy & Results

_This file tracks test methods, scenarios, and results with concrete execution evidence. Bugs found here block the release of a feature. Agents must update this during the Test and Fix phases._

## 0. Local Development Setup
_Exact steps to get the application running locally so agents can execute tests._

### Prerequisites
- Go 1.22+ installed.
- `golangci-lint` installed for linting.

### Start the App / Example
- Run the interactive CLI example: `go run ./cmd/example`

### Code Generation
- N/A

---

## 1. Test Methods & Tools

### Unit / Integration Tests
- **Run all tests**: `make test` (executes `go test -v -race ./...`).
- **Mocking**: The test suite validates memory safety limits and concurrency using standard library `net/http/httptest` mock servers. **No third-party mocking libraries are allowed.**
- **Parallelism**: Use parallel table-driven execution for test cases where applicable.

### Type Checking & Linting
- **Linting**: `make lint` (executes `golangci-lint run ./...`). This must produce 0 warnings.
- **Pre-commit**: Run `make setup` to configure `.githooks` which run linting automatically before commits.

## 2. Execution Evidence Rules
- For Go tests, paste the output of `make test` (or `go test -v -race ./...`) showing individual test PASS/FAIL lines in the Notes column.
- For type checking / linting, paste the output of `make lint` (must show no issues).
- "PASS" with no evidence is treated as UNTESTED.

---

## Regression Scenarios (Persistent)
_These scenarios survive the Ship phase cleanup. They are re-run on every release to catch regressions. Add critical paths and previously-shipped bug fixes here._

| Scenario | Last Verified | Notes |
|----------|---------------|-------|
| Race detector passes on all packages | _YYYY-MM-DD_ | Go: `go test -race ./...` |
| `golangci-lint` runs with zero warnings | _YYYY-MM-DD_ | Run `make lint` |
| HTTP mock server handles concurrent session accesses safely | _YYYY-MM-DD_ | Core validation for thread-safe cookiejar |
| Optional JSON fields parse cleanly to `nil` without panicking | _YYYY-MM-DD_ | Validates generic envelope unmarshaling |