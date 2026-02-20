# eero-go

[![Go Reference](https://pkg.go.dev/badge/github.com/arvind/eero-go.svg)](https://pkg.go.dev/github.com/arvind/eero-go)

`eero-go` is an unofficial, highly robust Go client library for interacting with the Eero Mesh Router API. 

Designed for performance and security, it seamlessly handles eero's two-step authentication, manages API cookie sessions thread-safely, and surfaces strongly-typed structs for your local network topology.

**Disclaimer:** This is an unofficial open-source library and is not affiliated with, endorsed by, or supported by Eero or Amazon.

## Features

- **No External Dependencies**: Built entirely upon the Go standard library (`net/http`, `context`, `encoding/json`).
- **Secure By Default**: Hardened `http.Transport` against connection exhaustion, 5MB `io.LimitReader` caps against OOM attacks, and strictly managed `cookiejar` states protecting session leaks.
- **Idiomatic Typings**: Extracts eero's deeply nested JSON envelopes into clean, traversable Go structs using Go 1.18+ Generics, dropping missing data fields safely to pointer `nil` values.

## Component Architecture

The library is modularized by functional domains to provide strict operational boundaries:

- `client.go`: Centralizes the HTTP `Client`, enforces security boundaries, limits payload sizes, and manages the thread-safe `net/http/cookiejar`.
- `auth.go`: Manages the undocumented 2-step verification challenge (Email/Phone -> OTP).
- `account.go`: Retrieves top-level user account details and base networking routing URLs.
- `network.go`: Safely parses Eero's `{"meta": {}, "data": {}}` JSON payloads to expose network operational status, exact port speeds, and health metrics.
- `device.go`: Lists all connected and recently offline devices, safely mapping absent optional fields (like IPs for offline devices) to `nil` using `*string` pointers.
- `profile.go`: Manages groupings of devices and offers the ability to pause/unpause internet blocks.

## System Workflow Diagram

```mermaid
flowchart TD
    User([User / CLI]) --> |Initiates| Client[Go Client Interface]
    
    Client --> CacheCheck{Check Local Session<br>.eero_session.json}
    
    CacheCheck -->|Exists & Valid| CookieJar[Inject Token into http.CookieJar]
    CacheCheck -->|Missing / Expired| AuthFlow[Interactive Auth Flow]
    
    AuthFlow -->|1. /login Email Request| EeroCloud[Eero Cloud API]
    AuthFlow -->|2. /login/verify 2FA Code| EeroCloud
    
    AuthFlow -.-> |Persist Token| CacheCheck
    
    CookieJar --> ServiceRequest[Execute Service Request<br>Account / Network / Device]
    
    ServiceRequest -->|Authenticated HTTP GET/POST| EeroCloud
    
    EeroCloud -->|Raw JSON Response| Unmarshal["Generic Envelope Unmarshaler<br>EeroResponse#91;T#93;"]
    Unmarshal -->|Type-Safe Go Structs| User
```

## Session Management

Authentication with Eero relies on a persistent session cookie (`s=user_token`). The `eero-go` client abstracts this away entirely:

1. Upon successful login, the token is transparently injected into the underlying `http.CookieJar`.
2. For long-running or local CLI tools, you can extract this `user_token` and save it locally (e.g., to a restrictive `0600` permission `.eero_session.json` file).
3. On subsequent boots, use `client.SetSessionCookie(token)` to instantly restore authorization without pinging users for another 2FA code.

## Installation

You need Go `1.21` or higher installed.

```bash
go get github.com/arvind/eero-go
```

## Quick Start

This snippet demonstrates initializing the client, generating an auth token from the CLI, and fetching network metrics.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/arvind/eero-go/eero"
)

func main() {
	// 1. Enforce a timeout against the entire flow
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 2. Initialize the client (secure cookie jar is built-in)
	client, err := eero.NewClient()
	if err != nil {
		log.Fatalf("failed to init client: %v", err)
	}

	// 3. Optional: Restore an existing session token from disk if you have one
	// client.SetSessionCookie("your_previously_saved_token")

	// 4. Interactive Login Flow (skip if you restored via SetSessionCookie)
	fmt.Print("Enter your eero email: ")
	var email string
	fmt.Scanln(&email)

	_, err = client.Auth.Login(ctx, email)
	if err != nil {
		log.Fatalf("login failed: %v", err)
	}

	fmt.Print("Check your email for the code: ")
	var code string
	fmt.Scanln(&code)

	if err := client.Auth.Verify(ctx, code); err != nil {
		log.Fatalf("verification failed: %v", err)
	}
	fmt.Println("Authenticated!")

	// 5. Fetch Account to find the Network URL mapping
	acct, err := client.Account.Get(ctx)
	if err != nil {
		log.Fatalf("fetching account failed: %v", err)
	}
	networkURL := acct.Networks.Data[0].URL

	// 6. Fetch Network Speeds
	net, err := client.Network.Get(ctx, networkURL)
	if err != nil {
		log.Fatalf("fetching network failed: %v", err)
	}
	
	fmt.Printf("Network Name: %s\n", net.Name)
	fmt.Printf("Download: %.1f %s\n", net.Speed.Down.Value, net.Speed.Down.Units)
}
```

## Testing

The test suite validates memory safety limits and concurrency using standard library `httptest` mock servers and parallel table execution. No third-party mocking libraries are strictly required!

```bash
# Run the test suite with the race detector
make test
```

## Building for Home Labs

The included `Makefile` handles cross-compilation for headless servers and minimal Linux environments (like Proxmox or TrueNAS).

```bash
# Compile for standard 64-bit Linux machines
make build-linux-amd64

# Compile for ARM servers (e.g., Raspberry Pi 4/5)
make build-linux-arm64

# Clean up build artifacts
make clean
```

## License

This project is licensed under the MIT License - see the `LICENSE` file for details.
