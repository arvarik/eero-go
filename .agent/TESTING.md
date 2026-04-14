# Testing Strategy & Execution Guidelines

_This file dictates exactly how testing is structured, executed, and verified across the `eero-go` SDK. The entire testing apparatus relies solely on Go standard library tooling — no third-party mocking, assertion, or comparison libraries._

## 1. Environment & Tooling

### Execution Commands

| Command | What It Does |
|---|---|
| `go test -v -race ./...` | Full test suite with verbose output and race detector |
| `golangci-lint run ./...` | Static analysis, unused variables, formatting |
| `make test` | Wraps `go test -v -race ./...` |
| `make lint` | Wraps `golangci-lint run ./...` |

### CI Pipeline (`.github/workflows/ci.yml`)

Triggered on every push/PR to `main`:
1. Ubuntu-latest runner with Go 1.21.
2. `go mod tidy` → `go test -v -race ./...` → `golangci-lint` (with `continue-on-error: true`, 5-minute timeout).

### Pre-Commit Hook (`.githooks/pre-commit`)

Runs `make lint` before every local commit. Blocks the commit if linting fails.

## 2. Test Architecture

### Test Package Conventions

The test suite uses two distinct package patterns, each with a specific purpose:

**External tests (`package eero_test`)** — Blackbox API-surface testing:
- Files: `client_test.go`, `client_session_test.go`, `auth_test.go`, `network_test.go`, `device_test.go`, `profile_test.go`, `errors_test.go`, `time_test.go`
- Import with: `"github.com/arvarik/eero-go/eero"`
- Can only access exported types and methods.
- Test the SDK exactly as a consumer would use it.

**Internal tests (`package eero`)** — Whitebox unit testing:
- Files: `client_internal_test.go`, `client_bench_test.go`
- No import needed — same package.
- Can access unexported fields and methods (`originURL()`, `newRequest()`, `newRequestFromURL()`).
- Used for testing internal URL construction, caching, and SSRF protection.

### `httptest` Server Mocking (No Third-Party Mocks)

Because `eero-go` is a networking SDK, tests NEVER contact the production Eero API. All tests use `net/http/httptest`:

**Standard Pattern:**
```go
// 1. Create a route-specific mux
mux := http.NewServeMux()
mux.HandleFunc("/2.2/networks/44444", func(w http.ResponseWriter, r *http.Request) {
    // Assert request constraints (method, headers, cookies)
    if r.Method != http.MethodGet {
        t.Errorf("Expected GET, got %s", r.Method)
    }
    
    // Return mock JSON matching Eero's envelope format
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write([]byte(`{
        "meta": {"code": 200, "server_time": "2023-10-01T12:00:00Z"},
        "data": {"name": "Home Mesh", "status": "online"}
    }`))
})

// 2. Spin up the httptest server
server := httptest.NewServer(mux)
defer server.Close()

// 3. Point the client to the mock
client, _ := eero.NewClient()
client.BaseURL = server.URL + "/2.2"  // Note: include version prefix for newRequestFromURL
```

**Alternative Pattern (simple handler):**
```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Single endpoint handler
}))
```

**BaseURL Configuration Rules:**
- For services using `newRequest()` (e.g., `AuthService`): Set `client.BaseURL = server.URL` (no version prefix).
- For services using `newRequestFromURL()` (e.g., `NetworkService`, `DeviceService`, `ProfileService`): Set `client.BaseURL = server.URL + "/2.2"` so the origin URL resolves correctly.

**Session Cookie Seeding in Tests:**
```go
// SetSessionCookie() enforces Secure: true, which blocks sending over http:// test URLs.
// Bypass by seeding the jar directly with a non-Secure cookie for testing:
testURL, _ := url.Parse(client.BaseURL)
client.HTTPClient.Jar.SetCookies(testURL, []*http.Cookie{
    {Name: "s", Value: "test_session_active"},
})
```

### Table-Driven Tests with Parallelization

Every test file uses structured table-driven tests with `t.Parallel()`:

```go
func TestServiceName_Method(t *testing.T) {
    t.Parallel()  // Top-level parallel

    tests := []struct {
        name         string
        mockStatus   int
        mockResponse string
        wantErr      bool
        // ... expected values
    }{
        {
            name:         "Success_DescriptiveCase",
            mockStatus:   http.StatusOK,
            mockResponse: `{"meta": {"code": 200}, "data": {...}}`,
            wantErr:      false,
        },
        {
            name:         "Failure_DescriptiveCase",
            mockStatus:   http.StatusNotFound,
            mockResponse: `{"meta": {"code": 404, "error": "Not found"}, "data": {}}`,
            wantErr:      true,
        },
    }

    for _, tc := range tests {
        tc := tc  // Capture range variable for parallel execution
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()  // Sub-test parallel

            // Setup mock server, create client, execute, assert
        })
    }
}
```

**Mandatory rules:**
- `t.Parallel()` on both the top-level test function AND each `t.Run()` subtest.
- `tc := tc` capture before the `t.Run()` closure to avoid data races on the loop variable.
- Each subtest spins up its own `httptest.NewServer` — no shared state between subtests.

### Pointer Safety Testing

The `device_test.go` file specifically validates that nil-pointer fields don't cause panics:

```go
// Helper functions used across tests:
func ptr(s string) *string     { return &s }                // Create pointer from literal
func equalStringPtr(a, b *string) bool { ... }              // Safe pointer comparison
func safeStr(s *string) string { if s == nil { return "<nil>" }; return *s }  // Safe logging
```

Tests verify:
- Online devices with all fields populated decode correctly.
- Offline devices with `null` or missing `nickname`, `ip` fields decode to `nil` pointers (not zero values).
- `VlanID` as `*int` correctly unmarshals.
- `EeroTime` custom timestamps parse in both Eero and RFC3339 formats.

### Error Type Assertion Testing

Tests verify `*APIError` surfaces correctly through `errors.As()`:

```go
var apiErr *eero.APIError
if !errors.As(err, &apiErr) {
    t.Fatalf("Expected error to be of type *eero.APIError, got %T", err)
}
if apiErr.HTTPStatusCode != 500 { ... }
if apiErr.Code != 500 { ... }
if apiErr.Message != "Internal Server Error" { ... }
```

The `errors_test.go` file additionally tests `IsAuthError()` against HTTP 401, API code 401, and non-401 cases.

## 3. Benchmark Infrastructure

### Internal Benchmarks (`eero/client_bench_test.go`)

Four benchmarks measuring JSON parsing performance of the dual deserialization paths:

| Benchmark | What It Measures |
|---|---|
| `BenchmarkDo` | `do()` path with valid large payload (1000 extra keys) |
| `BenchmarkDoRaw` | `doRaw()` path with valid large payload |
| `BenchmarkDoParseError` | `do()` path with malformed JSON |
| `BenchmarkDoRawParseError` | `doRaw()` path with malformed JSON |

Each benchmark spins up a local `httptest.NewServer` returning the pre-generated payload and measures end-to-end request + parse time.

### Internal Benchmark (`eero/client_internal_test.go`)

| Benchmark | What It Measures |
|---|---|
| `BenchmarkOriginURL` | Cache hit performance of `originURL()` double-checked locking |

### Root-Level Strategy Benchmark (`test_parse_bench_test.go`)

Compares two JSON parsing strategies at the module root level:

| Benchmark | Strategy |
|---|---|
| `BenchmarkDo_DoubleParse` | Two `json.Unmarshal` calls — once for meta, once for data |
| `BenchmarkDo_SingleParseRawMessage` | Single `json.Unmarshal` with `json.RawMessage` for data |

This benchmark validated the `json.RawMessage` approach used in `performRequestAndCheck()`. The baseline result is stored in `benchmark_baseline.txt`.

## 4. Complete Test File Inventory (12 Files)

| File | Package | Tests |
|---|---|---|
| `account_test.go` | `eero_test` | `TestAccountService_Get` (2 cases: full payload, internal server error) |
| `client_test.go` | `eero_test` | `TestLogin`, `TestGetNetwork`, `TestErrorHandling` |
| `client_session_test.go` | `eero_test` | `TestSetSessionCookie`, `TestSetSessionCookie_InvalidBaseURL` |
| `client_internal_test.go` | `eero` | `TestClient_originURL_Robustness`, `TestClient_newRequest_Concat`, `TestClient_newRequestFromURL_Resolve`, `TestClient_newRequestFromURL_SSRF`, `BenchmarkOriginURL` |
| `client_bench_test.go` | `eero` | `BenchmarkDo`, `BenchmarkDoRaw`, `BenchmarkDoParseError`, `BenchmarkDoRawParseError` |
| `account_test.go` | `eero_test` | `TestAccountService_Get` (2 cases: full payload, internal server error) |
| `auth_test.go` | `eero_test` | `TestAuthService_Login` (4 cases: success, invalid email, unauthorized, malformed JSON), `TestAuthService_Verify` (2 cases: success, rejection) |
| `network_test.go` | `eero_test` | `TestNetworkService_Get` (2 cases: full payload, not found), `TestNetworkService_Reboot` (2 cases: success, not found) |
| `device_test.go` | `eero_test` | `TestDeviceService_List` (2 cases: online/offline devices, bad gateway) |
| `profile_test.go` | `eero_test` | `TestProfileService_List` (2 cases), `TestProfileService_Pause`, `TestProfileService_Unpause` |
| `errors_test.go` | `eero_test` | `TestAPIError_Error` (2 cases), `TestAPIError_IsAuthError` (3 cases) |
| `time_test.go` | `eero_test` | `TestEeroTime_UnmarshalJSON` (5 cases: custom format, RFC3339, null, empty, invalid) |
| `test_parse_bench_test.go` | `main` | `BenchmarkDo_DoubleParse`, `BenchmarkDo_SingleParseRawMessage` |

## 5. Regression Matrix

_These scenarios MUST pass locally before any `git push` command is issued._

| Scenario | What It Validates | Test |
|---|---|---|
| **Pointer Null Safety** | Offline devices with `null` or missing `ip`/`nickname` fields decode to `nil` pointers — no panics | `go test ./eero -run TestDeviceService_List` |
| **EeroTime Parsing** | Custom format `+0000`, RFC3339 `Z`, `null`, and `""` all parse correctly | `go test ./eero -run TestEeroTime` |
| **SSRF Protection** | Cross-host and protocol-downgrade URLs are rejected by `newRequestFromURL()` | `go test ./eero -run TestClient_newRequestFromURL_SSRF` |
| **Session Cookie Security** | `Secure: true` cookie is NOT sent over HTTP URLs, only HTTPS | `go test ./eero -run TestSetSessionCookie` |
| **API Error Typing** | HTTP 500 responses surface as `*APIError` via `errors.As()` with correct fields | `go test ./eero -run TestErrorHandling` |
| **Auth Error Detection** | 401 responses trigger `IsAuthError() == true` | `go test ./eero -run TestAuthService_Login/Failure_Unauthorized` |
| **Race Detection** | All parallel tests pass under the race detector without data races | `go test -race ./eero` |
| **Code Quality** | No lint violations, no shadowed variables, no unused imports | `make lint` |

## 6. Execution Evidence Rules

Agents operating within test boundaries must conform exactly to these output mandates:

- If altering a test, agents MUST output logs showing `PASS` across both standard tests and the `-race` detector.
- If establishing a new model, the agent MUST output `make lint` confirming zero violations.
- **Silent passing is unacceptable.** You must copy the exact `go test` stdout to the active agent notes if executing changes manually.
- All new endpoint implementations MUST include at least two test cases: one success case and one error case (4xx or 5xx).