# Product Philosophy

_This document outlines the inviolable soul and core mission of `eero-go`. Engineers, Product Visionaries, and contributing AI Agents must continuously evaluate technical decisions against these core directives._

## 1. Why `eero-go` Exists
The Eero Mesh Router system operates on an entirely undocumented, constantly shifting secondary API originally designed solely for their iOS/Android applications. `eero-go` acts to bridge the gap for enterprise architectures and homelab enthusiasts by providing a profoundly stable, type-safe, and self-contained Go ecosystem. It flawlessly manages opaque two-step verifications, session caching, and generic JSON mapping without subjecting the consumer to dependency hell.

## 2. Who is the User?
- **Homelab Engineers**: Running `Proxmox` and `Raspberry Pi` clusters who need a tiny, statically-compiled binary to pipe local network health metrics (SNR margins, Node statuses) safely to external dashboards (like Grafana).
- **System Administrators**: Need automated background cron jobs that poll connectivity statuses without unexpectedly hitting API rate limits or facing recursive 2FA authentication walls. 
- **Internal Microservices**: Backend applications that require a headless context to execute macro controls (e.g., automated parental controls pausing profiles dynamically).

## 3. Core Directives

### The 'Zero-Dependency' Mandate
External libraries inevitably carry vulnerabilities, complex sub-trees, and eventual deprecation risks. `eero-go` operates completely isolated, exclusively bridging `net/http` and `encoding/json`. The minimal attack footprint ensures the codebase stays incredibly secure, lightning-fast to compile, and immune to broad dependency ecosystem changes.

### Secure By Default & Memory Hardened
The system is built recognizing that traversing internal hardware routing is deeply sensitive.
- We never hold unbounded network structures in memory. Eero's API responses are strictly constrained to 5MB caps, outright rejecting potential OOM memory leak attacks contextually.
- Session tokens are deliberately divorced from runtime logs or standard variables post-evaluation, enforcing strict OS-level `0600` permission architectures on local machine caches (`.eero_session.json`). The Token is solely entrusted to standard `http.CookieJar` state mechanics during program execution.

### Idiomatic Grace (Strict Typing) 
The upstream Cloud payload is erratic—keys suddenly drop on offline devices, string data maps poorly. The SDK must shield the consumer.
- We act as the sole interpreter. We rely critically on robust pointer mechanics (`*string`, `*time.Time`) parsing partial payloads seamlessly down to safely checked `nil` boundaries, avoiding brutal Go runtime panics on standard zero-value declarations.
- Error reporting utilizes typed standard nested structures mapping specifically back to Eero's cloud responses allowing upstream consumers cleanly wrapped contextual errors (`errors.As() / errors.Is()`).