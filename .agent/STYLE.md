# Style Guide & Code Conventions

_This document enforces the coding patterns specifically tailored for building and extending this zero-dependency Go SDK. It prevents context drift as multiple agents work on the codebase. Agents MUST follow these rules strictly when modifying any file within `eero/`._

## 1. Zero-Dependency Protocol

The `eero/` core package is strictly confined to Go standard library imports.

### Allowed Packages (`eero/` Core)

| Package | Usage |
|---|---|
| `bytes` | Response null checking, request body buffering |
| `context` | Request-scoped cancellation and timeouts |
| `encoding/json` | JSON marshaling/unmarshaling (request bodies, response envelopes) |
| `errors` | (Imported transitively — unused directly in current source, but permitted) |
| `fmt` | Error wrapping (`fmt.Errorf`), string formatting |
| `io` | `io.LimitReader`, `io.Reader`, `io.ReadAll` |
| `net/http` | HTTP client, requests, responses, cookie management |
| `net/http/cookiejar` | Thread-safe session cookie storage |
| `net/url` | URL parsing, resolution, origin extraction |
| `sync` | `sync.RWMutex` for origin URL cache concurrency |
| `time` | Transport timeouts, time parsing |

### Allowed Packages (`cmd/example/` CLI)

The CLI example may additionally import: `bufio`, `log`, `os`, `strings`, `text/tabwriter`.

### Allowed Packages (Tests)

Test files may additionally import: `testing`, `net/http/httptest`, `encoding/json`, `errors`, `io`, `net/url`.

### Forbidden Packages

> [!WARNING]
> Any `import` of the following in `eero/` core is an immediate rejection.

- Any `github.com/*` package.
- Any `golang.org/x/*` package.
- `testify`, `gomock`, `go-cmp`, `logrus`, `resty`, `cobra`, `viper`, or any third-party library.
- `io/ioutil` — fully deprecated since Go 1.16.

## 2. Syntax & Architecture Patterns

### Generic Envelope Deserialization

Eero's API wraps all data in a `{"meta": {...}, "data": ...}` envelope. Two deserialization strategies exist:

**Pattern 1: `doRaw()` with `EeroResponse[T]`** (preferred for most services):
```go
var resp EeroResponse[Account]
if err := s.client.doRaw(req, &resp); err != nil {
    return nil, fmt.Errorf("account: %w", err)
}
return &resp.Data, nil
```

**Pattern 2: `do()` with data-only extraction** (used by AuthService):
```go
var res LoginResponse
if err := s.client.do(req, &res); err != nil {
    return nil, err
}
```

**Rules:**
- NEVER manually parse the `meta`/`data` envelope inside service functions.
- ALWAYS use one of the two established patterns above.
- `doRaw()` is the standard path for read endpoints. `do()` is used when only the `data` portion is needed.

### Request Construction

Two request construction methods exist, each for a specific use case:

**`newRequest(ctx, serviceName, method, path, body)`** — For endpoints with fixed paths:
```go
// Path is appended to BaseURL via string concatenation
req, err := s.client.newRequest(ctx, "auth", http.MethodPost, "/login", body)
```

**`newRequestFromURL(ctx, serviceName, method, relativeURL, body)`** — For API-returned URLs:
```go
// relativeURL (e.g., "/2.2/networks/12345") is resolved against the origin
// Includes SSRF protection: host + scheme validation
req, err := s.client.newRequestFromURL(ctx, "network", http.MethodGet, networkURL, nil)
```

**Rules:**
- Use `newRequest()` for static paths (`/login`, `/login/verify`, `/account`).
- Use `newRequestFromURL()` for dynamic paths returned by the API (network URLs, profile URLs, device URLs).
- NEVER construct full URLs manually by string concatenation with API-returned paths.

### Context & Timeouts

- All service methods MUST accept `ctx context.Context` as the very first parameter.
- Pass this context directly into `http.NewRequestWithContext()` via `buildRequest()`.
- Do NOT build internal timeouts inside the SDK. Timeouts are exclusively the responsibility of the caller providing the context. The 30-second `http.Client.Timeout` exists only as a safety net.

### Error Handling & Typing

Always return strongly-typed errors that downstream applications can check against.

**Standard Library Wrapping:**
```go
return fmt.Errorf("eero: parsing base URL: %w", err)
```

**API-Level Errors:**
```go
// Returned automatically by performRequestAndCheck() for non-2xx or meta.code >= 400
apiErr := &combined.Meta
apiErr.HTTPStatusCode = statusCode
return nil, nil, apiErr
```

**Rules:**
- Use `fmt.Errorf("context: %w", err)` for wrapping — preserves the error chain.
- Never use bare `errors.New()` when wrapping an existing error.
- `*APIError` fulfills the `error` interface — consumers can use `errors.As(err, &apiErr)`.
- Always include the service name as error context prefix (e.g., `"account: %w"`, `"network: reboot: %w"`).

### JSON Pointer Semantics (Critical)

Eero's upstream API is undocumented and the response schema is unstable. When a device goes offline, fields like `ip`, `nickname`, or `manufacturer` may vanish entirely from the JSON payload rather than being set to `null` or a default.

**Mandate:**
- Any struct field mapped via `json:"xyz"` that could potentially be omitted MUST use a pointer type:
  - `*string`, `*int`, `*bool`, `*time.Time`, `*int64`
- Currently pointer-typed fields in `Device`: `Manufacturer`, `IP`, `Nickname`, `Hostname`, `Usage`, `VlanID`, `DisplayName`, `ModelName`, `ManufacturerDeviceTypeID`
- Currently pointer-typed fields in `DeviceConnectivity.RateInfo`: `RateBps`, `MCS`, `NSS`, `GuardInterval`, `ChannelWidth`, `PhyType`

**Forbidden:**
- NEVER use bare `string`, `int`, or `bool` for fields that aren't guaranteed to always be present.
- NEVER use `json:"xyz,omitempty"` as a substitute for pointer semantics — `omitempty` controls marshaling, not unmarshaling.

### HTTP Method Usage

| Method | When | Example |
|---|---|---|
| `http.MethodGet` | Read-only data retrieval | `Account.Get()`, `Network.Get()`, `Device.List()`, `Profile.List()` |
| `http.MethodPost` | Authentication, destructive actions | `Auth.Login()`, `Auth.Verify()`, `Network.Reboot()` |
| `http.MethodPut` | Idempotent state updates | `Profile.Pause()`, `Profile.Unpause()` |

## 3. Naming Conventions

### File Topology

| Pattern | Purpose | Example |
|---|---|---|
| `domain.go` | Domain service + models | `network.go`, `device.go`, `profile.go` |
| `domain_test.go` | External blackbox tests (`package eero_test`) | `network_test.go`, `device_test.go` |
| `client_test.go` | External client integration tests | `client_test.go`, `client_session_test.go` |
| `client_internal_test.go` | Internal whitebox tests (`package eero`) | `client_internal_test.go` |
| `client_bench_test.go` | Internal benchmarks (`package eero`) | `client_bench_test.go` |

### Variable / Func Casing

- `camelCase` for unexported internal variables and helper functions (e.g., `originMu`, `cachedOriginURL`, `setPaused`, `buildRequest`).
- `PascalCase` for all exported API interfaces and data structs (e.g., `NetworkDetails`, `DeviceService`, `SetSessionCookie`).
- `json:"snake_case"` strictly for struct tags mapping to upstream Eero Cloud keys (e.g., `json:"user_token"`, `json:"ip_address"`).
- **Package Name:** `eero` — strictly lowercase, single word.

### Struct Naming Patterns

- **Service structs**: `{Domain}Service` (e.g., `AuthService`, `NetworkService`).
- **Response models**: Descriptive nouns (e.g., `NetworkDetails`, `Device`, `Profile`, `Account`).
- **Nested sub-models**: `{Parent}{Child}` (e.g., `NetworkConnection`, `DeviceConnectivity`, `AccountPhone`).
- **Request bodies**: `{Action}Request` for exported (e.g., `LoginRequest`, `VerifyRequest`), `camelCase` for unexported (e.g., `pauseRequest`).

## 4. Anti-Patterns (FORBIDDEN)

> [!CAUTION]
> DO NOT merge code containing these patterns.

- ❌ **NEVER** read unbounded HTTP response bodies. Always use `io.LimitReader(resp.Body, 5*1024*1024)`.
- ❌ **NEVER** use `interface{}` or `any` for JSON unmarshaling where a generic envelope or concrete struct can be defined.
- ❌ **NEVER** use standard zero-value struct fields for optional Eero JSON blocks. Use pointers.
- ❌ **NEVER** construct HTTP requests that bypass the `client.HTTPClient` and its `CookieJar`.
- ❌ **NEVER** ignore the Go linting rules (`golangci-lint run ./...`). Pre-commit hooks will block it.
- ❌ **NEVER** create a service method without `ctx context.Context` as the first parameter.
- ❌ **NEVER** use `ioutil.ReadAll` — it is deprecated. Use `io.ReadAll` with `io.LimitReader`.
- ❌ **NEVER** set the session cookie via `req.Header.Set("Cookie", ...)` — always use `SetSessionCookie()` → `CookieJar`.
- ❌ **NEVER** use `url.Parse` + string concatenation to build URLs for API-returned paths. Use `newRequestFromURL()`.