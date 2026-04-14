# Architecture

_This document acts as the definitive anchor for understanding system design, data models, API contracts, and technology boundaries. Update this document during the Design and Review phases._

## 1. Tech Stack & Infrastructure
- **Language / Runtime**: Go 1.22+
- **Frontend**: N/A (SDK library and CLI tool)
- **Backend / API**: Go standard library (`net/http`, `context`, `encoding/json`). Zero third-party dependencies for the core library.
- **Database**: N/A — Interacts with Eero Cloud API. Local session cached in `.eero_session.json`.
- **Deployment**: Distributed as a Go module (`go get github.com/arvarik/eero-go`). Example CLI can be built via `make build-local` or cross-compiled.
- **Package Management**: Go modules (`go.mod`) + Makefile.
- **Build System**: `go build` via Makefile.

## 2. System Boundaries & Data Flow

### Request / Data Flow
- **Authentication Flow**: User / CLI Initiates → Checks local `.eero_session.json` → If exists, injects token into `http.CookieJar`. If missing, interactive Auth Flow (Email/Phone → OTP) → Persists token to `.eero_session.json`.
- **Service Request Flow**: `http.CookieJar` handles authenticated sessions → Execute Service Request (Account, Network, Device) → Authenticated HTTP GET/POST to Eero Cloud API → Raw JSON Response → Generic Envelope Unmarshaler (`EeroResponse[T]`) → Type-Safe Go Structs returned to User.
- **Data Parsing**: Extracts Eero's deeply nested JSON envelopes into clean, traversable Go structs using Go Generics, dropping missing data fields safely to pointer `nil` values.

### Concurrency / Threading Model
- **Thread Safety**: Strictly managed `cookiejar` states protect against session leaks across concurrent requests.
- **HTTP Transport**: Hardened `http.Transport` against connection exhaustion.

## 3. Data Models & Database Schema
N/A — No database utilized. Exposes Eero's cloud data models.

## 4. API Contracts

The core library is modularized by functional domains:
- **Client (`client.go`)**: Centralizes the HTTP `Client`, enforces security boundaries, limits payload sizes (5MB `io.LimitReader`), and manages the thread-safe `net/http/cookiejar`.
- **Auth (`auth.go`)**: Manages the undocumented 2-step verification challenge (Email/Phone -> OTP).
- **Account (`account.go`)**: Retrieves top-level user account details and base networking routing URLs.
- **Network (`network.go`)**: Safely parses deeply nested JSON payloads to expose granular telemetry (IPv6Leases, DHCP allocations, PremiumDNS) and network health metrics.
- **Device (`device.go`)**: Exhaustively lists all connected and offline devices with detailed connectivity reporting, mapping absent optional fields to `nil` using `*string` pointers.
- **Profile (`profile.go`)**: Manages groupings of fully mapped Device topologies and pausing internet blocks.

## 5. External Integrations / AI
- **Eero Cloud API**: Interacts with undocumented Eero endpoints (`https://api-user.e2ro.com/2.2/`).
- **Session Caching**: `.eero_session.json` is used to cache the user token locally to bypass 2FA challenges on subsequent runs. Must be stored with restrictive `0600` permissions.

## 6. Invariants & Safety Rules
- NEVER introduce external third-party dependencies to the core library. Only the Go standard library is allowed.
- ALWAYS use `io.LimitReader` (capped at 5MB) to protect against OOM attacks when reading HTTP responses.
- ALWAYS manage session tokens using the thread-safe `net/http/cookiejar`.
- NEVER panic on missing optional JSON fields. ALWAYS map them to pointer `nil` values.
- ALWAYS cache the session token in a file with strict `0600` permissions (`.eero_session.json`).

## 7. Error Handling Patterns
- Explicit `if err != nil` error propagation.
- Custom error types defined in `errors.go` mapping to specific Eero API failure states.

## 8. Directory Structure
- `cmd/` — Example CLI application entrypoints (`cmd/example/main.go`).
- `eero/` — Core SDK library services and domain models.
- `.githooks/` — Custom git hooks configured via `make setup`.

## 9. Local Development
- **Install / Setup**: Run `make setup` to configure local git hooks.
- **Build**: `make build-local` (or `make build-linux-amd64` / `make build-linux-arm64`).
- **Test**: `make test` runs tests with the race detector.
- **Lint**: `make lint` runs `golangci-lint`.
- **Run Example CLI**: `go run ./cmd/example`

## 10. Environment Variables
N/A — Configuration and authentication is handled interactively or via `.eero_session.json`.