# Architecture

_This document acts as the definitive anchor for understanding system design, data models, API contracts, and technology boundaries. Agents must update this document critically when introducing new network data models or modifying structural dependencies._

## 1. Tech Stack & Infrastructure
- **Language / Runtime**: Go 1.22+
- **Architecture Profile**: Zero-dependency Headless Go SDK & reference interactive CLI.
- **Backend / Core Networking**:
  - Exclusively standard library: `net/http`, `context`, `encoding/json`.
  - Concurrency managed internally via `net/http/cookiejar`.
- **Local Persistence layer**: `os/user` to handle `0600` persisted cache `.eero_session.json` token.
- **Distribution Model**: Modular go package (`github.com/arvarik/eero-go`).
- **Build System**: Managed extensively by `Makefile` (handles cross-compiled homelab architectures like `amd64` and `arm64`).

## 2. System Boundaries & Execution Flow

### State Management & Authentication Flow
- Eero utilizes a persistent session cookie (`s=user_token`) and a strict rate-limited challenge-response architecture.
- **Cold Boot Flow**: System verifies `.eero_session.json`. If missing, `auth.go` triggers `/login` -> email/SMS verification -> `/login/verify` -> injects secure cookie into global `http.CookieJar`.
- **Warm Boot Flow**: `client.go` statically injects the parsed token from `.eero_session.json` into the thread-safe `CookieJar`, entirely bypassing the Auth API, protecting against Eero rate-limit blacklisting.

### Core Execution Flow
- Authenticated `http.Client` context → Request scoped by `context.Context` (allowing caller cancellation/timeouts) → Endpoint target generated via base host constant.
- Returned `*http.Response` is intercepted and strictly piped through an `io.LimitReader(resp.Body, 5*1024*1024)` to safeguard against OOM memory leak attacks from malformed Eero upstream data.
- Payload unmarshaled seamlessly using `EeroResponse[T]any` typed generic wrappers mapping deep arbitrary JSON directly to structured data.

## 3. Modular Functional Domains (`eero/*.go`)

This project avoids monolithic client files. All APIs strictly reside in domain-specific components chained to the `eero.Client{}` struct:
- **`client.go`**: State container. Enforces 5MB Payload limits, TLS verification hardening, and strictly acts as the gateway via its internal `*http.Client`.
- **`auth.go`**: Orchestrates Interactive 2-Step Verification and gracefully surfaces credential errors.
- **`account.go`**: Traverses top-level account IDs and maps dynamic base API routing strings.
- **`network.go`**: Evaluates physical topography (Gateway routers, Leaf nodes) and telemetry (Bandwidth Down/Up).
- **`device.go`**: Meticulously handles generic mapping of clients vs. offlined nodes, explicitly resolving missing fields (IP, MACs, active status) into `*string/int` pointers preventing Go zero-value panic faults.
- **`profile.go`**: Orchestrates external logical grouping mechanisms (User network pausing).
- **`errors.go`**: Intercepts generic `40X/50X` response codes and morphs them into typed `eero.APIError` structures, preventing string-matched error hacking in consumer clients.
- **`time.go`**: Handles custom/proprietary date formats shipped by Eero Cloud into standard `time.Time`.

## 4. Invariants & Safety Mandates (Critical for AI Agents)
> [!WARNING]
> DO NOT DEVIATE from these standard practices when modifying `eero-go`.

1. **NO EXTERNAL DEPENDENCIES:** Adding third-party packages to `eero/` core is an absolute anti-pattern. Everything must be mapped by `std` libraries.
2. **5MB LIMITERS:** Any new endpoints added MUST respect the `io.LimitReader`. DO NOT process Unbounded memory reads, `ioutil.ReadAll` is fully deprecated.
3. **THREAD-SAFE SESSIONING:** The `s=cookie` must NEVER be loaded manually as an `http.Header`. Always utilize the established runtime `CookieJar` attached to the transport client.
4. **POINTER NULL MAPPING:** JSON `omitempty` is insufficient for deeply nested structures. All arbitrarily null values from the Eero API must map cleanly to `*type` struct elements. Zero representations fall on the consumer, not the SDK parser.

## 5. Development Layout
- **`cmd/example/main.go`**: Integration layer demonstrating cold/warm authentication and iterative domain calls using the underlying `eero` package.
- **`eero/`**: Restricted SDK boundary.
- **`.githooks/`**: Executable pre-commit checks natively binding `make lint` prior to upstream progression. Ensures 100% compliant commits natively locally.