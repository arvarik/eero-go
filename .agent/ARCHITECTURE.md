# Architecture

_This document is the definitive anchor for understanding system design, data models, API contracts, security boundaries, and technology infrastructure. Agents must update this document critically when introducing new endpoints, modifying structural dependencies, or altering security invariants._

## 0. Project Topology

**Topology:** `[library-sdk, backend]`

_Agents: Read the corresponding Gemstack topology profiles (`library-sdk.md` and `backend.md`) from `~/.gemini/antigravity/global_workflows/` before proceeding with any workflow step. These profiles enforce API surface stability, backward compatibility, zero-dependency discipline, data integrity testing, and anti-mocking rules._

## 1. Tech Stack & Infrastructure

- **Language / Runtime**: Go 1.22+ (`go.mod` declares `go 1.22.0`; CI runs against Go 1.21 for broad compatibility).
- **Architecture Profile**: Zero-dependency headless Go SDK with a reference interactive CLI (`cmd/example/main.go`).
- **Module Path**: `github.com/arvarik/eero-go`
- **Standard Library Only**: The `eero/` package exclusively uses `std` — no third-party imports exist anywhere in `go.mod`.
- **Build System**: `Makefile` providing `tidy`, `test`, `lint`, `build-local`, `build-linux-amd64`, `build-linux-arm64`, `clean`, and `setup` targets.
- **CI/CD**: GitHub Actions (`.github/workflows/ci.yml`) runs on `ubuntu-latest` — `go mod tidy`, `go test -v -race ./...`, and `golangci-lint` on every push/PR to `main`.
- **License**: MIT.

## 2. Repository File Tree

```
eero-go/
├── .agent/                          # AI agent documentation suite (this directory)
│   ├── ARCHITECTURE.md
│   ├── PHILOSOPHY.md
│   ├── STATUS.md
│   ├── STYLE.md
│   └── TESTING.md
├── .github/
│   ├── ISSUE_TEMPLATE/              # Bug report & feature request templates
│   └── workflows/ci.yml            # GitHub Actions CI pipeline
├── .githooks/
│   └── pre-commit                   # Runs `make lint` before every commit
├── cmd/
│   └── example/
│       └── main.go                  # Interactive CLI demonstrating full SDK flow
├── docs/
│   ├── archive/                     # Historical design documents (empty, .gitkeep)
│   ├── designs/                     # Struct binding designs (empty, .gitkeep)
│   ├── explorations/                # Raw API exploration notes (empty, .gitkeep)
│   └── plans/                       # Implementation plans (empty, .gitkeep)
├── eero/                            # Core SDK package — zero external dependencies
│   ├── client.go                    # HTTP client, transport, security, request factory
│   ├── auth.go                      # Two-step login/verify authentication
│   ├── account.go                   # Account details & network URL discovery
│   ├── network.go                   # Network topology, eero nodes, speed, health, reboot
│   ├── device.go                    # Connected/offline device listing with pointer safety
│   ├── profile.go                   # User profiles with pause/unpause internet control
│   ├── errors.go                    # Typed APIError struct implementing `error` interface
│   ├── time.go                      # EeroTime custom JSON unmarshaler for non-RFC3339 dates
│   ├── *_test.go                    # Comprehensive test suite (see TESTING.md)
│   └── *_bench_test.go             # Benchmark suite for JSON parsing paths
├── .eero_session.json               # Local session cache (gitignored, 0600 permissions)
├── .gitignore
├── benchmark_baseline.txt           # Stored benchmark baseline output
├── go.mod                           # Module definition (zero require directives)
├── LICENSE                          # MIT License
├── Makefile                         # Build, test, lint, cross-compile targets
├── README.md                        # Public-facing documentation
└── test_parse_bench_test.go         # Root-level benchmark: double-parse vs single-parse
```

## 3. Core Client Architecture (`eero/client.go`)

The `Client` struct is the central orchestrator for all API interactions. It manages HTTP transport, session cookies, URL construction, and response processing.

### 3.1 Client Struct Fields

| Field | Type | Purpose |
|---|---|---|
| `HTTPClient` | `*http.Client` | Handles requests with cookie jar, custom transport, redirect policy |
| `BaseURL` | `string` | Root API URL, default `https://api-user.e2ro.com/2.2` |
| `UserAgent` | `string` | Spoofed User-Agent, default `eero/3.0 (iPhone; iOS 17.0)` |
| `Auth` | `*AuthService` | Two-step authentication service |
| `Account` | `*AccountService` | Account details & network discovery |
| `Network` | `*NetworkService` | Network topology, telemetry, reboot |
| `Device` | `*DeviceService` | Client device listing |
| `Profile` | `*ProfileService` | User profiles, pause/unpause |
| `originMu` | `sync.RWMutex` | Protects `cachedOriginURL` / `originURLSnapshot` |
| `cachedOriginURL` | `*url.URL` | Cached scheme+host origin for URL resolution |
| `originURLSnapshot` | `string` | BaseURL snapshot for cache invalidation |

### 3.2 Custom HTTP Transport

`NewClient()` configures a hardened `http.Transport` instead of relying on `http.DefaultTransport`:

```go
transport := &http.Transport{
    Proxy:                 http.ProxyFromEnvironment,
    ForceAttemptHTTP2:     true,
    MaxIdleConns:          100,
    MaxIdleConnsPerHost:   10,
    IdleConnTimeout:       90 * time.Second,
    TLSHandshakeTimeout:  10 * time.Second,
    ExpectContinueTimeout: 1 * time.Second,
}
```

The `http.Client` itself has a **30-second fallback timeout** and a **`CheckRedirect`** policy that:
- **Blocks cross-domain redirects** — prevents open-redirect session hijacking.
- **Caps redirects at 10** — prevents infinite redirect loops.

### 3.3 Request Construction (Dual Paths)

| Method | Usage | URL Strategy |
|---|---|---|
| `newRequest()` | Simple endpoints (e.g., `/login`, `/account`) | String concatenation: `BaseURL + path` |
| `newRequestFromURL()` | Endpoints from API-returned URLs (e.g., `/2.2/networks/12345`) | `url.ResolveReference()` against `originURL()` with **SSRF protection** |

Both converge in `buildRequest()`, which:
1. Marshals the body to JSON if non-nil.
2. Calls `http.NewRequestWithContext()` — all requests carry a `context.Context`.
3. Sets `User-Agent` and `Content-Type` headers.

### 3.4 SSRF & Protocol Downgrade Protection

`newRequestFromURL()` enforces that the resolved URL's **host** and **scheme** match the configured API origin:

```go
if u.Host != base.Host || u.Scheme != base.Scheme {
    return nil, fmt.Errorf("eero: security policy blocked request to %s://%s (expected %s://%s)", ...)
}
```

This prevents:
- Requests to attacker-controlled hosts via manipulated API URLs.
- Protocol downgrades from HTTPS to HTTP that could leak the session cookie.

### 3.5 Response Processing (Dual Deserialization Paths)

| Method | Strategy | Used By |
|---|---|---|
| `do(req, v)` | Two-pass: parse `meta`+`data` as `json.RawMessage`, then unmarshal `data` into `v` | `AuthService` |
| `doRaw(req, v)` | Single-pass: unmarshal full body into `EeroResponse[T]` | `AccountService`, `NetworkService`, `DeviceService`, `ProfileService` |

Both share a common `performRequestAndCheck()` layer that:
1. Reads the body via `io.LimitReader(resp.Body, 5*1024*1024)` — **5MB hard cap**.
2. Unmarshals the `meta` envelope and checks for error codes.
3. Returns a typed `*APIError` for any non-2xx status or `meta.code >= 400`.

### 3.6 Origin URL Caching

`originURL()` extracts scheme+host from `BaseURL` using a **double-checked locking** pattern:
- **Fast path** (`RLock`): Returns cached origin if `BaseURL` hasn't changed.
- **Slow path** (`Lock`): Re-parses and updates the cache.
- Returns a **copy** to prevent callers from mutating the cached value.

### 3.7 Generic Envelope Type

```go
type EeroResponse[T any] struct {
    Meta APIError `json:"meta"`
    Data T        `json:"data"`
}
```

Used with concrete type parameters (e.g., `EeroResponse[Account]`, `EeroResponse[[]Device]`) to provide compile-time type safety for API responses.

## 4. Authentication Flow (`eero/auth.go`)

The Eero API uses an undocumented two-step challenge-response:

1. **`Login(ctx, identifier)`** → `POST /login` with `{"login": "email_or_phone"}` → Returns `user_token`, automatically sets session cookie via `SetSessionCookie()`.
2. **`Verify(ctx, code)`** → `POST /login/verify` with `{"code": "123456"}` → Activates the session.

### Session Cookie Management (`SetSessionCookie`)

```go
c.HTTPClient.Jar.SetCookies(u, []*http.Cookie{{
    Name: "s", Value: userToken, Secure: true, HttpOnly: true,
}})
```

- **`Secure: true`**: Cookie only transmitted over HTTPS — prevents interception over HTTP.
- **`HttpOnly: true`**: Prevents client-side script access to the session token.
- The `cookiejar` is **thread-safe** — safe for concurrent goroutine access.

## 5. Modular Functional Domains (`eero/*.go`)

### `account.go` — AccountService

- **`Get(ctx)`** → `GET /account` → Returns `Account` struct.
- **Key Data**: User name, email, phone, `Networks.Data` containing `NetworkSummary` entries with `.URL` fields (e.g., `/2.2/networks/12345`) used as input for downstream services.
- **Rich Model**: Maps 15+ nested structs including `PremiumDetails`, `PushSettings`, `Consents`, `AccountAuth`.

### `network.go` — NetworkService

- **`Get(ctx, networkURL)`** → `GET {networkURL}` → Returns `NetworkDetails`.
- **`Reboot(ctx, networkURL)`** → `POST {networkURL}/reboot` → Triggers full network reboot.
- **Key Data**: Network name, status, WAN IP, DHCP/DNS config, speed tests (`SpeedMeasurement`), health indicators, guest network, premium DNS/adblocking, firmware updates, IPv6 config.
- **`EeroNode` struct**: Maps individual mesh hardware with serial, model, IP, firmware, mesh quality, connected client count, heartbeat, IPv6 addresses, power info, and bands.
- **Uses `EeroTime`** for the `Joined` field on `EeroNode`.

### `device.go` — DeviceService

- **`List(ctx, networkURL)`** → `GET {networkURL}/devices` → Returns `[]Device`.
- **Pointer-Safe Design**: Fields that the API may omit for offline devices use `*string`, `*int`, `*bool` pointers — `Nickname`, `IP`, `Manufacturer`, `Hostname`, `Usage`, `VlanID`, `DisplayName`, `ModelName`, `ManufacturerDeviceTypeID`.
- **Rich Connectivity Data**: `DeviceConnectivity` with `RateInfo` (rx/tx bitrates, MCS, NSS, guard interval, channel width, PHY type), `EthernetStatus`, signal metrics.
- **Uses `EeroTime`** for `LastActive` and `FirstActive` fields.

### `profile.go` — ProfileService

- **`List(ctx, networkURL)`** → `GET {networkURL}/profiles` → Returns `[]Profile`.
- **`Pause(ctx, profileURL)`** → `PUT {profileURL}` with `{"paused": true}` — Blocks internet.
- **`Unpause(ctx, profileURL)`** → `PUT {profileURL}` with `{"paused": false}` — Restores internet.
- **Key Data**: Profile name, paused state, device count, full `[]Device` array, safe search, block apps, optional `Schedule` bedtime.

### `errors.go` — Typed Error System

```go
type APIError struct {
    HTTPStatusCode int    `json:"-"`       // HTTP response status code
    Code           int    `json:"code"`    // API-level meta.code
    Message        string `json:"error"`   // Human-readable error
    ServerTime     string `json:"server_time"`
}
```

- Implements `error` interface via `func (e *APIError) Error() string`.
- **`IsAuthError()`**: Returns `true` if `HTTPStatusCode == 401 || Code == 401`.
- Enables `errors.As(err, &apiErr)` for downstream type assertion by consumers.

### `time.go` — EeroTime Custom Unmarshaler

```go
type EeroTime struct { time.Time }
```

- Implements `json.Unmarshaler` to handle Eero's non-standard timestamp format `2006-01-02T15:04:05Z0700` (e.g., `+0000` without colon).
- **Parsing strategy**: Try custom format first → fallback to `time.RFC3339` → error.
- **Handles edge cases**: JSON `null` → zero time, empty string `""` → zero time, fast-path quoted string extraction.

## 6. CLI Reference (`cmd/example/main.go`)

The example CLI demonstrates the full SDK lifecycle:

1. **Session Restoration**: Reads `.eero_session.json` → injects token via `SetSessionCookie()`.
2. **Session Validation**: Calls `Account.Get()` to verify the token is still valid.
3. **Auth Fallback**: If token expired → `IsAuthError()` check → falls through to interactive `Login()` + `Verify()`.
4. **Session Persistence**: Writes `{"user_token": "..."}` to `.eero_session.json` with `0600` permissions.
5. **Data Display**: Fetches `Account` → extracts `networkURL` → fetches `NetworkDetails` → lists `[]Device` → prints with `tabwriter`.

## 7. Invariants & Safety Mandates (Critical for AI Agents)

> [!CAUTION]
> These are inviolable. Violating any of these produces an immediate PR rejection.

1. **ZERO EXTERNAL DEPENDENCIES**: The `eero/` package must import only Go `std` libraries. No `github.com/*`, no `golang.org/x/*`. Zero entries in `go.mod` `require` block.
2. **5MB PAYLOAD CAP**: Every response body read must use `io.LimitReader(resp.Body, 5*1024*1024)`. Never use `io.ReadAll(resp.Body)` without the limiter. `ioutil.ReadAll` is fully deprecated.
3. **THREAD-SAFE SESSIONS**: The session cookie (`s=user_token`) must **only** be set via `client.SetSessionCookie()` which routes through the `http.CookieJar`. Never set it as a raw `http.Header`.
4. **POINTER NULL MAPPING**: Any struct field mapped from Eero JSON that could be omitted **must** be a pointer type (`*string`, `*int`, `*bool`, `*time.Time`). Non-pointer types for optional fields are forbidden.
5. **CONTEXT ON EVERY REQUEST**: All service methods must accept `ctx context.Context` as the first parameter and pass it to `http.NewRequestWithContext()`.
6. **SSRF GUARD**: Never bypass the host/scheme validation in `newRequestFromURL()`. All API-returned URLs must flow through this method.
7. **NO REDIRECT LEAKS**: The `CheckRedirect` policy must remain — never set `CheckRedirect: nil`.

## 8. Build & CI Infrastructure

### Makefile Targets

| Target | Command | Purpose |
|---|---|---|
| `tidy` | `go mod tidy && go fmt ./...` | Clean deps + format |
| `test` | `go test -v -race ./...` | Full test suite with race detector |
| `lint` | `golangci-lint run ./...` | Static analysis |
| `build-local` | `go build -o bin/eero-go ./cmd/example` | Local binary |
| `build-linux-amd64` | `GOOS=linux GOARCH=amd64 go build ...` | x86_64 cross-compile |
| `build-linux-arm64` | `GOOS=linux GOARCH=arm64 go build ...` | ARM64 cross-compile |
| `clean` | `rm -rf bin/ && rm -f .eero_session.json` | Remove artifacts |
| `setup` | `git config core.hooksPath .githooks` | Install pre-commit hooks |

### Pre-Commit Hook (`.githooks/pre-commit`)

Runs `make lint` before every commit. Blocks the commit if linting fails. Bypass with `git commit --no-verify` (not recommended).

### GitHub Actions CI (`.github/workflows/ci.yml`)

Triggered on push/PR to `main`:
1. Checkout → Setup Go 1.21 → `go mod tidy` → `go test -v -race ./...` → `golangci-lint`.
2. Lint step uses `continue-on-error: true` with 5-minute timeout.