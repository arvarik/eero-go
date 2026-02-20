# eero-go

[![Go Reference](https://pkg.go.dev/badge/github.com/arvind/eero-go.svg)](https://pkg.go.dev/github.com/arvind/eero-go)

`eero-go` is an unofficial, highly robust Go client library for interacting with the Eero Mesh Router API. 

Designed for performance and security, it seamlessly handles eero's two-step authentication, manages API cookie sessions thread-safely, and surfaces strongly-typed structs for your local network topology.

**Disclaimer:** This is an unofficial open-source library and is not affiliated with, endorsed by, or supported by Eero or Amazon.

## Features

- **No External Dependencies**: Built entirely upon the Go standard library (`net/http`, `context`, `encoding/json`).
- **Secure By Default**: Hardened `http.Transport` against connection exhaustion, 5MB `io.LimitReader` caps against OOM attacks, and strictly managed `cookiejar` states protecting session leaks.
- **Idiomatic Typings**: Extracts eero's nested JSON envelopes into clean, traversable Go structs, dropping missing data fields safely to pointer `nil` values.

## Available Services

- `Auth`: Handles the email/phone challenge (`Login()`) and 2FA code submission (`Verify()`).
- `Account`: Retrieves user account details and base networking routing URLs.
- `Network`: Checks network operational status, exact port speeds, and triggers reboots.
- `Device`: Lists all connected and recently offline devices, including IPs, MAC addresses, and active timestamps.
- `Profile`: Manages groupings of devices and offers the ability to pause/unpause internet bounds.

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

## License

This project is licensed under the MIT License - see the `LICENSE` file for details.
