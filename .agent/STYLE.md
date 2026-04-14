# Style Guide & Code Conventions

_This document enforces the visual identity and coding patterns of the project. It prevents context drift as multiple agents work on the codebase. Agents MUST follow these rules strictly._

## 1. Visual Language & Tokens
N/A — SDK / backend-only project.

## 2. Component Patterns
N/A — SDK / backend-only project.

## 3. Code Conventions

### Architecture Patterns
- **Zero-Dependency SDK**: The core library in `eero/` MUST NOT rely on any external Go packages. All functionality must be built on top of the standard library (`net/http`, `context`, `encoding/json`).
- **Generic Envelope Unmarshaling**: Use Go 1.18+ Generics (`EeroResponse[T]`) to unmarshal Eero's deeply nested JSON envelopes into clean, traversable Go structs.
- **Modular Service Design**: Services are split into discrete domains (`auth.go`, `account.go`, `network.go`, `device.go`, `profile.go`) attached to a core `Client`.

### State Management
- **Session Management**: Session state (the authentication cookie) is managed internally by `net/http/cookiejar`. For persistence across runs, the token is extracted and saved to `.eero_session.json`.
- **Stateless Services**: The domain services (`Account`, `Network`, etc.) are stateless and rely on the `Client` to provide the authenticated HTTP transport.

### Strict Typing
- All code MUST pass `golangci-lint run ./...` with zero warnings.
- Optional fields in JSON responses MUST be represented as pointers (e.g., `*string`, `*int`) so that absent data correctly unmarshals to `nil` rather than zero-values.

## 4. Naming Conventions
- **Files**: `snake_case.go` for Go files (e.g., `account_test.go`, `client_session_test.go`).
- **Variables / Functions**: `camelCase` for unexported internal variables/functions, `PascalCase` for exported public API surface.
- **Structs / Types**: `PascalCase` for all exported data models.
- **Packages**: `eero` is a lowercase, single-word package name.

## 5. Import Ordering
- Standard `goimports` ordering:
  1. Standard library (`fmt`, `net/http`, `context`, `encoding/json`).
  2. (No third-party dependencies allowed in the core library).

## 6. Documentation Standards
- **Docstrings**: Provide `godoc`-compatible comments on all exported types and functions (`// FunctionName does X.`).

## 7. Anti-Patterns (FORBIDDEN)
- ❌ NEVER introduce third-party dependencies outside the standard library to the `eero/` package.
- ❌ NEVER read unbounded HTTP response bodies. Always use `io.LimitReader` (e.g., capped at 5MB).
- ❌ NEVER use `interface{}` / `any` for JSON unmarshaling where a strongly typed generic or concrete struct can be used.
- ❌ NEVER represent optional JSON fields as concrete zero-values (use pointers like `*string` to handle `nil`).
- ❌ NEVER bypass the `http.CookieJar` for session token management during active requests.
- ❌ NEVER disable `golangci-lint` checks.