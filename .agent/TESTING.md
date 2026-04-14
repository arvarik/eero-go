# Testing Strategy & Execution Guidelines

_This file dictates exactly how testing is structured, executed, and verified across the `eero-go` SDK without injecting bloated third-party mocking libraries._

## 1. Environment and Primitives
_The entire testing apparatus relies solely on the Go `cmd/go` standard spec._
- **Execution Engine:** `go test -v -race ./...`
- **Linting Engine:** `golangci-lint run ./...`
- **Local Helper:** The `Makefile` wraps both: `make test` and `make lint`.

## 2. Test Architecture Methods

### `httptest` Server Mocking (No Third-Party Mocks)
Because `eero-go` acts as a networking SDK, we NEVER actually ping the production Eero API during unit tests. 
- Agents must spin up local `httptest.NewServer()` mocks bridging a custom router that matches the requested request path (`/2.2/account`).
- Return raw JSON strings simulating Eero Cloud responses (both 200 OK and 4xx/5xx payloads).
- Overwrite the internal `eero.Client{}` domain host address natively mapping it to the `httptest` ephemeral url.

### Table-Driven Test Parallelization
Tests must use structurally defined slice tables and iterate using sub-tests:
```go
tests := []struct {
    name       string
    setupMock  func() *httptest.Server
    assertFunc func(t *testing.T, res Result, err error)
}{ ... }

for _, tc := range tests {
    t.Run(tc.name, func(t *testing.T) {
        // execute
    })
}
```

### The `cookiejar` Concurrency Matrix
Because `net/http/cookiejar` is injected into the global `http.Client` transport, agents MUST ensure testing aggressively captures Session leaks.
- All tests evaluating `auth.go` or session states MUST dynamically trigger parallelism and run the `-race` detector to guarantee memory is strictly fenced.

## 3. Strict Execution Evidence Rules
Agents operating within Test boundaries must conform exactly to these output mandates. Bugs found here block PR merges.
- If altering a test, agents must output logs showing `PASS` across both standard tests and the `-race` detector execution.
- If establishing a new model, the agent must output `make lint` confirming zero cyclic dependencies and unused vars.
- **Silent passing is unacceptable. You must copy the exact `go test` standard out to the active agent notes if executing changes manually on the developer's machine.**

---

## 4. End-to-End Regression Matrix
_These scenarios survive active phase cleanups. They must pass locally before any `git push` command is issued._

| Scenario | Assertion Layer | Method |
|----------|-----------------|--------|
| **Data Dropping Tolerance** | Validate that JSON returning offline Eero units where IP fields (`ipv4`, `ipv6`) are entirely stripped from the payload DO NOT panic the `Device` structs and correctly map to pointer `nil`. | `go test ./eero` |
| **OOM Boundary Limits** | Attempt to return a >5MB payload string via `httptest` mock. Assert that `io.LimitReader` successfully detonates and chokes the ingestion gracefully returning bounds errors instead of crashing the process tree. | `go test ./eero -run TestClient_HitPayloadLimit` |
| **Race Verification** | Spin 10 parallel goroutines attempting to `/login/verify` an OTP while polling `client.Device.List()`. Assert the underlying mutex lock inside `cookiejar` handles the parallel HTTP transports flawlessly. | `go test -race ./eero` |
| **Code Structure Integrity** | Assert that `go imports -w` formatting is strictly adhered to and no shadowed variables exist. | `make lint` |