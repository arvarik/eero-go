# Style Guide & Code Conventions

_This document enforces the coding patterns specifically tailored for building this strict Go SDK. It prevents context drift as multiple agents work on the codebase. Agents MUST follow these rules strictly._

## 1. Zero-Dependency Protocol
The `eero/` core package is strictly confined.
- **Allowed Packages:** Core `std`, primarily `net/http`, `context`, `time`, `bytes`, `fmt`, `encoding/json`, `io`, and `errors`.
- **Forbidden Packages:** Any external library (e.g., `github.com/*` or `golang.org/x/*`) including `logrus`, `resty`, `testify`, or `go-cmp`.

## 2. Syntax and Architecture Patterns

### Generic Envelope Deserialization
Eero's API wraps data indiscriminately. E.g., `{"meta": {...}, "data": [ ... ]}`.
- NEVER map these manually inside service functions.
- ALWAYS use the `eero.EeroResponse[T]` generic envelope for unmarshaling API responses. Send the inner concrete struct as the generic parameter `T`.

### Context & Timeouts
- All service methods (`Account.Get()`, `Network.Reboot()`) MUST accept a `ctx context.Context` as the very first parameter.
- Pass this context directly into the `http.NewRequestWithContext` constructor. 
- Do not build internal timeouts inside the SDK. Timeouts are exclusively the responsibility of the caller providing the context.

### Error Handling & Typing
Always return strongly-typed errors that downstream applications can check against.
- Standard Library Wrapping: Use `fmt.Errorf("contextual message: %w", err)` to wrap standard library errors, not `errors.New`.
- Eero Specific Failures: Use the internally defined `eero.APIError` to represent upstream 4xx/5xx status codes natively.

### JSON Pointer Semantics (Critical) 
Eero's upstream API is notoriously undocumented. If a device goes offline, boolean variables might vanish entirely from the JSON rather than safely reverting to `false`.
- **Mandate:** Any struct field mapped via `json:"xyz"` that could potentially be omitted from the Eero payload MUST be mapped as a pointer (`*string`, `*int`, `*bool`).
- **Forbidden:** Never use `string`, `int`, or `bool` for fields that aren't guaranteed, doing so causes runtime zero-value defaults that misrepresent actual hardware telemetry.

## 3. Naming Conventions

### File Topology
- `domain.go` (e.g., `network.go`, `device.go`).
- `domain_test.go` for blackbox external API testing (e.g., `network_test.go`).
- `domain_internal_test.go` for internal package state assertions.

### Variable / Func Casing
- `camelCase` for unexported internal variables and helper functions.
- `PascalCase` for all exported API interfaces and telemetry Data Structs.
- `json:"snake_case"` strictly for struct tags mapping to upstream Eero Cloud keys.
- **Package Name:** `eero` (strictly lowercase, single word).

## 4. Anti-Patterns (FORBIDDEN)
> [!WARNING]
> DO NOT MERGE CODE containing these patterns.

- ❌ NEVER read unbounded HTTP response bodies. **Always use bounded reads** (e.g., `io.LimitReader(resp.Body, 5*1024*1024)`).
- ❌ NEVER use `interface{}` or `any` for JSON unmarshaling where a generic envelope or concrete struct can be defined.
- ❌ NEVER instantiate standard zero-values struct fields for missing Eero Optional JSON blocks.
- ❌ NEVER construct HTTP Requests that bypass the `client.cookieJar`.
- ❌ NEVER ignore the Go Linting rules (`golangci-lint run ./...`). Build tasks will fail immediately on pre-commit hooks.