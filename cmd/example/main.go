// Command example demonstrates a complete interactive flow with the eero-go
// client library. It implements:
//
//   - Local session caching via .eero_session.json (0600 permissions)
//   - Strict context timeouts on every API call
//   - Graceful fallback from cached session to interactive login
//   - Tabwriter-formatted device listing with safe pointer dereferencing
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/arvarik/eero-go/eero"
)

// sessionFile is the local path where the session token is cached.
const sessionFile = ".eero_session.json"

// sessionData is the JSON structure persisted to disk.
type sessionData struct {
	UserToken string `json:"user_token"`
}

func main() {
	// Enforce a hard deadline on the entire program execution.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	// ── 1. Initialize the client ────────────────────────────────────────
	client, err := eero.NewClient()
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	// ── 2. Attempt to restore a cached session ──────────────────────────
	if err := restoreSession(client); err != nil {
		// No cached session (or file unreadable) — fall through to login.
		fmt.Println("No cached session found; starting interactive login.")
		if err := interactiveLogin(ctx, client); err != nil {
			return fmt.Errorf("login flow: %w", err)
		}
	} else {
		// Validate the cached token by hitting a lightweight endpoint.
		fmt.Println("Restored cached session. Validating…")
		if _, err := client.Account.Get(ctx); err != nil {
			// Token was rejected (expired / revoked). Fall back to login.
			var apiErr *eero.APIError
			if errors.As(err, &apiErr) && apiErr.IsAuthError() {
				fmt.Println("Cached session expired; re-authenticating.")
				if err := interactiveLogin(ctx, client); err != nil {
					return fmt.Errorf("login flow: %w", err)
				}
			} else {
				return fmt.Errorf("validating session: %w", err)
			}
		} else {
			fmt.Println("Session is valid.")
		}
	}

	// ── 3. Fetch account details ────────────────────────────────────────
	acct, err := client.Account.Get(ctx)
	if err != nil {
		return fmt.Errorf("fetching account: %w", err)
	}
	fmt.Printf("\n── Account ──\n")
	fmt.Printf("Name:  %s\n", acct.Name)
	fmt.Printf("Email: %s\n", acct.Email.Value)

	if acct.Networks.Count == 0 {
		fmt.Println("No networks found on this account.")
		return nil
	}

	// Extract the full relative URL for the first network.
	// This is passed directly to downstream services — no manual path construction.
	networkURL := acct.Networks.Data[0].URL
	fmt.Printf("Primary network URL: %s\n", networkURL)

	// ── 4. Fetch network details ────────────────────────────────────────
	net, err := client.Network.Get(ctx, networkURL)
	if err != nil {
		return fmt.Errorf("fetching network: %w", err)
	}
	fmt.Printf("\n── Network ──\n")
	fmt.Printf("Name:   %s\n", net.Name)
	fmt.Printf("Status: %s\n", net.Status)
	fmt.Printf("Speed:  ↓ %.1f %s  ↑ %.1f %s\n",
		net.Speed.Down.Value, net.Speed.Down.Units,
		net.Speed.Up.Value, net.Speed.Up.Units,
	)

	// ── 5. List connected devices ───────────────────────────────────────
	devices, err := client.Device.List(ctx, networkURL)
	if err != nil {
		return fmt.Errorf("listing devices: %w", err)
	}

	fmt.Printf("\n── Devices (%d) ──\n", len(devices))
	printDeviceTable(devices)

	return nil
}

// ─── Session Management ─────────────────────────────────────────────────────

// restoreSession reads the cached user_token from disk and injects it into the
// client's cookie jar.
func restoreSession(client *eero.Client) error {
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		return fmt.Errorf("reading session file: %w", err)
	}

	var sess sessionData
	if err := json.Unmarshal(data, &sess); err != nil {
		return fmt.Errorf("parsing session file: %w", err)
	}
	if sess.UserToken == "" {
		return fmt.Errorf("session file contains empty token")
	}

	// Inject the token into the client's cookie jar so all subsequent
	// requests carry the Cookie: s=<user_token> header.
	return client.SetSessionCookie(sess.UserToken)
}

// saveSession writes the user_token to disk with strict 0600 permissions
// so that only the file owner can read or modify it.
func saveSession(userToken string) error {
	sess := sessionData{UserToken: userToken}
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}
	if err := os.WriteFile(sessionFile, data, 0600); err != nil {
		return fmt.Errorf("writing session file: %w", err)
	}
	return nil
}

// ─── Interactive Login ──────────────────────────────────────────────────────

// interactiveLogin drives the two-step email → verification-code flow,
// prompting the user on stdin.
func interactiveLogin(ctx context.Context, client *eero.Client) error {
	reader := bufio.NewReader(os.Stdin)

	// Step 1: Send the login challenge.
	fmt.Print("Enter your eero email or phone: ")
	identifier, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading identifier: %w", err)
	}
	identifier = strings.TrimSpace(identifier)

	loginResp, err := client.Auth.Login(ctx, identifier)
	if err != nil {
		return fmt.Errorf("initiating login: %w", err)
	}
	fmt.Println("Verification code sent to your device.")

	// Step 2: Verify the code.
	fmt.Print("Enter verification code: ")
	code, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading verification code: %w", err)
	}
	code = strings.TrimSpace(code)

	if err := client.Auth.Verify(ctx, code); err != nil {
		return fmt.Errorf("verifying code: %w", err)
	}
	fmt.Println("Authenticated successfully!")

	// Persist the session token so we skip login next time.
	if err := saveSession(loginResp.UserToken); err != nil {
		// Non-fatal — warn but continue.
		fmt.Fprintf(os.Stderr, "warning: could not cache session: %v\n", err)
	} else {
		fmt.Printf("Session cached to %s\n", sessionFile)
	}

	return nil
}

// ─── Output Formatting ─────────────────────────────────────────────────────

// printDeviceTable renders the device slice as an aligned table using
// text/tabwriter. Pointer fields that may be nil are safely dereferenced
// with a fallback of "N/A".
func printDeviceTable(devices []eero.Device) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 3, ' ', 0)
	fmt.Fprintln(w, "NICKNAME\tMAC ADDRESS\tIP ADDRESS\tSTATUS")
	fmt.Fprintln(w, "--------\t-----------\t----------\t------")

	for _, d := range devices {
		nickname := deref(d.Nickname, "N/A")
		ip := deref(d.IP, "N/A")
		status := "offline"
		if d.Connected {
			status = "online"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", nickname, d.MAC, ip, status)
	}

	w.Flush()
}

// deref safely dereferences a *string, returning fallback if the pointer is nil.
func deref(s *string, fallback string) string {
	if s != nil {
		return *s
	}
	return fallback
}
