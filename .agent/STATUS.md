# eero-go Status

Last updated: 2026-04-14

_This file tracks the operational state, build milestones, API coverage, and pending actions. It is the definitive source of truth for "where am I?" AI Agents MUST update this file comprehensively after adding new endpoints, restructuring data models, or completing milestone tasks._

## Current Context

The core SDK is **fully operational** with complete coverage of the five primary Eero API domains: Auth, Account, Network, Device, and Profile. The test suite passes with race detection enabled, all security invariants are enforced, and CI/CD runs on every push/PR. The `.agent/` documentation suite has been hardened to production-grade standards.

## API Coverage Inventory

### Services & Methods

| Service | Method | HTTP | Endpoint | Returns |
|---|---|---|---|---|
| `AuthService` | `Login(ctx, identifier)` | `POST` | `/login` | `*LoginResponse` |
| `AuthService` | `Verify(ctx, code)` | `POST` | `/login/verify` | `error` |
| `AccountService` | `Get(ctx)` | `GET` | `/account` | `*Account` |
| `NetworkService` | `Get(ctx, networkURL)` | `GET` | `{networkURL}` | `*NetworkDetails` |
| `NetworkService` | `Reboot(ctx, networkURL)` | `POST` | `{networkURL}/reboot` | `error` |
| `DeviceService` | `List(ctx, networkURL)` | `GET` | `{networkURL}/devices` | `[]Device` |
| `ProfileService` | `List(ctx, networkURL)` | `GET` | `{networkURL}/profiles` | `[]Profile` |
| `ProfileService` | `Pause(ctx, profileURL)` | `PUT` | `{profileURL}` | `error` |
| `ProfileService` | `Unpause(ctx, profileURL)` | `PUT` | `{profileURL}` | `error` |

### Core Client Methods

| Method | Scope | Purpose |
|---|---|---|
| `NewClient()` | Exported | Factory — creates client with hardened transport, cookie jar, security policies |
| `SetSessionCookie(token)` | Exported | Injects token into cookie jar (`Secure: true`, `HttpOnly: true`) |
| `newRequest()` | Internal | Build request via string concatenation for static paths |
| `newRequestFromURL()` | Internal | Build request via URL resolution with SSRF protection |
| `buildRequest()` | Internal | Shared request factory (body marshaling, headers, context) |
| `performRequest()` | Internal | Execute request + read body with 5MB `io.LimitReader` |
| `performRequestAndCheck()` | Internal | Parse meta envelope + error detection |
| `do()` | Internal | Two-pass deserialization — extract `data` via `json.RawMessage` |
| `doRaw()` | Internal | Single-pass deserialization — full `EeroResponse[T]` |
| `originURL()` | Internal | Cache origin (scheme+host) with double-checked locking |

### Data Model Count

| Domain | Exported Structs |
|---|---|
| `client.go` | `Client`, `EeroResponse[T]` |
| `auth.go` | `AuthService`, `LoginRequest`, `LoginResponse`, `VerifyRequest` |
| `account.go` | `AccountService`, `Account`, `AccountEmail`, `AccountPhone`, `AccountNetworks`, `NetworkSummary`, `AccountAuth`, `ReportIssue`, `PremiumDetails`, `PushSettings`, `Consents`, `MarketingEmailsConsent` |
| `network.go` | `NetworkService`, `NetworkDetails`, `NetworkConnection`, `GeoIP`, `NetworkLease`, `LeaseDHCP`, `NetworkDHCP`, `NetworkDNS`, `DNSParent`, `NetworkSpeed`, `NetworkTimezone`, `NetworkUpdates`, `GuestNetwork`, `IPSettings`, `PremiumDNS`, `DNSPolicies`, `AdBlockSettings`, `NetworkPremiumDetails`, `IPv6Lease`, `NetworkIPv6`, `NetworkEeros`, `SpeedMeasurement`, `Health`, `InternetHealth`, `HealthDetail`, `EeroNode`, `IPv6Address`, `PowerInfo` |
| `device.go` | `DeviceService`, `Device`, `DeviceRef`, `DeviceSource`, `Usage`, `DeviceConnectivity`, `RateInfo`, `EthernetStatus`, `DeviceInterface`, `Homekit`, `RingLTE` |
| `profile.go` | `ProfileService`, `Profile`, `Schedule` |
| `errors.go` | `APIError` |
| `time.go` | `EeroTime` |

## Build & CI Status

| System | Status | Notes |
|---|---|---|
| `make test` | ✅ Passing | Race detector enabled |
| `make lint` | ✅ Passing | golangci-lint |
| GitHub Actions CI | ✅ Passing | Push/PR to `main`, Go 1.21, ubuntu-latest |
| Pre-commit hook | ✅ Active | Runs `make lint` on every commit |
| Cross-compilation | ✅ Working | `linux/amd64` and `linux/arm64` targets |

## Recently Completed

- [x] Core SDK implementation: Auth, Account, Network, Device, Profile services
- [x] Security hardening: SSRF protection, open-redirect guards, 5MB payload caps, Secure cookies
- [x] Custom `EeroTime` JSON unmarshaler for non-RFC3339 timestamps
- [x] Comprehensive test suite: 12 test files, 16+ test functions, 7 benchmarks
- [x] CI/CD pipeline: GitHub Actions with race detector and golangci-lint
- [x] Pre-commit hooks: `make lint` enforcement via `.githooks/pre-commit`
- [x] Example CLI: Interactive authentication with session caching
- [x] Cross-compilation Makefile targets for homelab deployment
- [x] `.agent/` documentation suite hardened to production-grade standards

## Known Limitations

- **Read-Heavy API Surface**: The SDK currently supports `PUT` only for `Profile.Pause()`/`Unpause()` and `POST` for `Auth.Login()`/`Verify()` and `Network.Reboot()`. Arbitrary `PUT`/`POST` operations on device nicknames, network settings, etc. are not yet implemented.
- **No Pagination**: Device and profile list endpoints return all results in a single response. If Eero adds pagination to these endpoints in the future, the SDK will need cursor/offset support.
- **Single Network Focus**: The example CLI operates on the first network in the account. Multi-network orchestration is left to the consumer.
- **CI Lint `continue-on-error`**: The golangci-lint step in CI uses `continue-on-error: true` — lint failures don't block the pipeline. This should be tightened for production.

## Development Lifecycle (Adding New Endpoints)

To add a new Eero API endpoint, agents should follow this sequential process:

1. **Exploratory Payload Fetch**: Document raw cURL responses in `docs/explorations/YYYY-MM-DD-endpoint-analysis.md`.
2. **Model Definition**: Design Go structs with pointer semantics in `docs/designs/YYYY-MM-DD-endpoint-structs.md`.
3. **Core Implementation**: Add the method to the appropriate domain `.go` file in `eero/`.
4. **Test Implementation**: Write table-driven tests with `httptest` mocks — minimum 1 success + 1 error case.
5. **Run `make test`**: All tests must pass with `-race` flag.
6. **Run `make lint`**: Zero violations.
7. **Update `STATUS.md`**: Add the new method to the API Coverage Inventory above.
8. **Update `ARCHITECTURE.md`**: Document the new domain method in Section 5.

## Roadmap

| Task | Priority | Type | Status |
|---|---|---|---|
| Add `Network.SpeedTest()` — trigger on-demand speed test | Medium | New Endpoint | Not Started |
| Add `Device.Get()` — single device detail fetch | Medium | New Endpoint | Not Started |
| Add `Device.Rename()` — update device nickname | Low | Mutative Endpoint | Not Started |
| Add `Network.UpdateDNS()` — configure DNS settings | Low | Mutative Endpoint | Not Started |
| Tighten CI lint to `continue-on-error: false` | Medium | DevOps | Not Started |
| Add `Device.Block()` / `Unblock()` — blacklist control | Low | Mutative Endpoint | Not Started |