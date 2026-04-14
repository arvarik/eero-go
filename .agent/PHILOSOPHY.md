# Product Philosophy

_This is the soul of the product. It explains why the app exists and what its core beliefs are. Product Visionaries and UI/UX Designers use this to make feature and design decisions. Engineers use it to resolve ambiguity._

## 1. Why This Exists
There is a need for a highly robust, secure, and performant Go client library to interact with the unofficial Eero Mesh Router API. This library seamlessly handles eero's two-step authentication, manages API cookie sessions thread-safely, and surfaces strongly-typed structs for local network topology without relying on bloated external dependencies.

## 2. Target User
This is for homelab enthusiasts, system administrators, and developers building automated cron jobs, local network monitoring tools, or home automation integrations who need a reliable, zero-dependency Go SDK to poll their Eero routers.

## 3. Core Beliefs
- **Zero External Dependencies**: The core library must rely entirely on the Go standard library. This ensures maximum portability, minimal attack surface, and long-term stability.
- **Secure By Default**: The library must implement strict memory safety boundaries (like `io.LimitReader` caps to prevent OOM attacks) and rigorous session management (thread-safe cookie jars, restrictive file permissions) to protect local network credentials.
- **Idiomatic and Strongly Typed**: Deeply nested, undocumented JSON APIs are messy. The SDK must abstract this mess away, presenting a clean, traversable, and strictly typed interface to the developer, safely handling missing data.

## 4. Design & UX Principles
- **Developer Ergonomics**: Authentication, session caching, and complex API responses should be abstracted into simple service methods (`client.Account.Get()`, `client.Network.Get()`).
- **Resilience**: The client should handle connection exhaustion gracefully with a hardened `http.Transport` and clearly propagate typed errors.

## 5. What This Is NOT
- Not an official Amazon or Eero product.
- Not a UI application. This is a headless SDK and command-line reference implementation.
- Not a bloated framework. It is a focused, single-purpose client library.