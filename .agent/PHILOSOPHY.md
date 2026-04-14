# Product Philosophy

_This document outlines the inviolable soul and core mission of `eero-go`. Engineers, Product Visionaries, and contributing AI Agents must continuously evaluate all technical decisions against these core directives. This philosophy drives every architectural choice documented in `ARCHITECTURE.md`._

## 1. Why `eero-go` Exists

The Eero Mesh Router system operates on an entirely undocumented, constantly shifting API originally designed solely for their iOS/Android mobile applications. There is no officially supported SDK, no public API documentation, and no stable contract.

`eero-go` exists to bridge this gap by providing a **profoundly stable, type-safe, and self-contained Go ecosystem** that:
- Flawlessly manages opaque two-step verification flows.
- Caches sessions locally to avoid punitive rate-limiting.
- Maps erratic, inconsistent JSON payloads into idiomatic Go structs using pointer semantics and generic envelopes.
- Operates without a single third-party dependency — compiling to a small, statically-linked binary suitable for constrained environments.

## 2. Who is the User?

### Homelab Engineers
Running `Proxmox`, `TrueNAS`, and `Raspberry Pi` clusters who need a tiny, statically-compiled binary to pipe local network health metrics (SNR margins, node statuses, speed tests) safely to external dashboards (like Grafana). They cross-compile for `linux/amd64` and `linux/arm64` via the Makefile and deploy as cron jobs.

### System Administrators
Need automated background monitoring that polls connectivity statuses without unexpectedly hitting API rate limits or facing recursive 2FA authentication walls. The session caching mechanism (`0600`-permissioned `.eero_session.json`) is specifically designed for this unattended use case.

### Internal Microservices
Backend applications that require a headless context to execute macro controls — such as automated parental controls pausing profiles dynamically via `Profile.Pause()` / `Profile.Unpause()`, or triggering network-wide reboots via `Network.Reboot()`.

### SDK Consumers
Go developers importing `github.com/arvarik/eero-go/eero` as a library dependency, leveraging the type-safe interfaces to build their own tooling without inheriting transitive dependency trees.

## 3. Core Directives

### The 'Zero-Dependency' Mandate

External libraries inevitably carry vulnerabilities, complex sub-trees, and eventual deprecation risks. `eero-go` operates completely isolated from the broader Go module ecosystem.

**What this means concretely:**
- `go.mod` has zero `require` directives — exclusively `module` and `go` declarations.
- The `eero/` package imports only `std` packages: `bytes`, `context`, `encoding/json`, `fmt`, `io`, `net/http`, `net/http/cookiejar`, `net/url`, `sync`, `time`, and `errors`.
- The `cmd/example/` CLI additionally uses `bufio`, `log`, `os`, `strings`, and `text/tabwriter` — all standard library.
- Testing uses `testing`, `net/http/httptest` — also standard library.

**Why this matters:**
- Minimal attack surface — no supply-chain vulnerabilities from transitive dependencies.
- Lightning-fast compilation — no dependency graph to resolve.
- Immune to ecosystem breaking changes (`left-pad` scenarios).
- Trivial to audit — the entire codebase is self-contained.

### Secure By Default & Defense-in-Depth

The system is built recognizing that traversing internal home/enterprise network infrastructure is deeply sensitive. Security is layered, not bolted on:

**Memory Safety — 5MB Payload Caps:**
- Every HTTP response body is read through `io.LimitReader(resp.Body, 5*1024*1024)`. This outright prevents OOM memory exhaustion from malformed or malicious upstream responses. The limit applies to every single endpoint uniformly through the centralized `performRequest()` method.

**Session Token Isolation:**
- Session tokens are stored exclusively in the `http.CookieJar` during runtime — never held in raw string variables post-injection.
- The `SetSessionCookie()` method sets `Secure: true` (HTTPS-only transit) and `HttpOnly: true` (no script access).
- The local file cache (`.eero_session.json`) is written with strict `0600` OS-level permissions — only the file owner can read or modify it.

**SSRF & Open-Redirect Protection:**
- `newRequestFromURL()` validates that the resolved URL's host AND scheme match the configured API origin before sending any request. This prevents Server-Side Request Forgery attacks via manipulated API-returned URLs.
- The `CheckRedirect` callback on `http.Client` blocks cross-domain redirects entirely, preventing open-redirect session hijacking where a malicious server could lure the cookie jar into leaking the `s=` session token to an attacker-controlled domain.
- Redirect chains are capped at 10 hops to prevent infinite loops.

**Transport Hardening:**
- A custom `http.Transport` is configured with explicit connection pool limits (`MaxIdleConns: 100`, `MaxIdleConnsPerHost: 10`), timeouts (`IdleConnTimeout: 90s`, `TLSHandshakeTimeout: 10s`), and HTTP/2 negotiation (`ForceAttemptHTTP2: true`). This prevents unbounded idle connection exhaustion and hanging TLS handshakes.
- A 30-second fallback client timeout caps the total duration of any HTTP exchange.

**User-Agent Spoofing:**
- The client sends `User-Agent: eero/3.0 (iPhone; iOS 17.0)` — mimicking the official Eero iOS app. This is a deliberate strategy: the Eero API is designed exclusively for their mobile apps and may reject or behave differently for unrecognized user agents. This spoofing ensures API compatibility.

### Idiomatic Grace — Strict Typing Over Silent Failure

The upstream Eero Cloud payload is erratic — keys suddenly vanish on offline devices, string data maps poorly, timestamp formats deviate from RFC3339. The SDK must shield the consumer from this chaos.

**Pointer-Based Null Safety:**
- Every struct field that could potentially be omitted from the Eero JSON response is mapped as a pointer type (`*string`, `*int`, `*bool`, `*time.Time`). This ensures that missing JSON keys decode to `nil` rather than Go's zero values (`""`, `0`, `false`), which would misrepresent actual hardware telemetry.
- Zero-value interpretation is the **consumer's** responsibility, not the SDK's. The SDK reports exactly what Eero sent — nothing more, nothing less.

**Custom Time Handling:**
- The `EeroTime` type implements `json.Unmarshaler` to handle Eero's non-standard timestamp format (`2006-01-02T15:04:05+0000`) which lacks the RFC3339-required colon in the timezone offset. The parser tries the custom format first, then falls back to `time.RFC3339`, and handles `null` and empty string edge cases gracefully.

**Typed Error Surfaces:**
- Errors are never raw strings. The `APIError` struct captures both the HTTP status code and the API-level `meta.code` + `meta.error` fields.
- `APIError` implements the `error` interface, enabling `errors.As(err, &apiErr)` type assertions by downstream consumers. The `IsAuthError()` helper specifically identifies 401 authentication failures.
- Standard library errors are wrapped with `fmt.Errorf("context: %w", err)` preserving the full error chain.

**Generic Envelope Deserialization:**
- The `EeroResponse[T]` generic struct provides compile-time type safety for API response unmarshaling. Rather than using `interface{}` or `any` and runtime type assertions, the concrete data type is specified at the call site (e.g., `EeroResponse[Account]`, `EeroResponse[[]Device]`).
- The dual deserialization strategy (`do()` extracts `data` via `json.RawMessage`; `doRaw()` unmarshals the full envelope) provides flexibility while maintaining type safety across different API response patterns.

## 4. What We Will Never Do

> [!WARNING]
> These anti-patterns are categorically forbidden across the entire codebase.

- ❌ **Add external dependencies** to the `eero/` package or `go.mod`.
- ❌ **Read unbounded response bodies** without `io.LimitReader`.
- ❌ **Use non-pointer types** for optional Eero JSON fields.
- ❌ **Bypass the `CookieJar`** by manually setting session cookies as HTTP headers.
- ❌ **Skip `context.Context`** on any method that performs I/O.
- ❌ **Suppress errors** — every error must be returned or wrapped, never silently discarded.
- ❌ **Use `ioutil.ReadAll`** — fully deprecated since Go 1.16.
- ❌ **Use `interface{}` or `any`** for JSON unmarshaling where a generic envelope or concrete struct can be defined.