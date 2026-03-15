// get-auth0-token.go - CLI tool to fetch Auth0 access tokens for API testing
//
// Usage:
//   go run tools/get-auth0-token.go -email user@example.com -password secret123
//
// Or with environment variables:
//   export AUTH0_DOMAIN=dev-xxxx.us.auth0.com
//   export AUTH0_CLIENT_ID=your_client_id
//   export AUTH0_AUDIENCE=https://api.howsworkingthere.com
//   export AUTH0_EMAIL=user@example.com
//   export AUTH0_PASSWORD=secret123
//   go run tools/get-auth0-token.go
//
// Output:
//   {"access_token":"eyJ...","token_type":"Bearer","expires_in":86400}
//
// To use in Postman:
//   1. Run this tool and copy the access_token value
//   2. Paste into Postman collection variable {{auth_token}}
//
// Note: This uses Auth0's Resource Owner Password Flow, which must be enabled
// in your Auth0 application settings (Advanced → Grant Types → Password).

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

type auth0TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope,omitempty"`
	IDToken     string `json:"id_token,omitempty"`
}

type auth0Error struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func main() {
	var (
		domain    = flag.String("domain", getEnv("AUTH0_DOMAIN", ""), "Auth0 domain (e.g., dev-xxxx.us.auth0.com)")
		clientID  = flag.String("client-id", getEnv("AUTH0_CLIENT_ID", ""), "Auth0 application client ID")
		audience  = flag.String("audience", getEnv("AUTH0_AUDIENCE", ""), "Auth0 API audience identifier")
		email     = flag.String("email", getEnv("AUTH0_EMAIL", ""), "User email")
		password  = flag.String("password", getEnv("AUTH0_PASSWORD", ""), "User password")
		realm     = flag.String("realm", getEnv("AUTH0_REALM", "Username-Password-Authentication"), "Auth0 database connection name")
		grantType = flag.String("grant-type", "http://auth0.com/oauth/grant-type/password-realm", "OAuth grant type")
	)
	flag.Parse()

	// Validate required flags
	missing := []string{}
	if *domain == "" {
		missing = append(missing, "domain (AUTH0_DOMAIN)")
	}
	if *clientID == "" {
		missing = append(missing, "client-id (AUTH0_CLIENT_ID)")
	}
	if *audience == "" {
		missing = append(missing, "audience (AUTH0_AUDIENCE)")
	}
	if *email == "" {
		missing = append(missing, "email (AUTH0_EMAIL)")
	}
	if *password == "" {
		missing = append(missing, "password (AUTH0_PASSWORD)")
	}

	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "Missing required parameters:\n")
		for _, m := range missing {
			fmt.Fprintf(os.Stderr, "  - %s\n", m)
		}
		fmt.Fprintf(os.Stderr, "\nUsage:\n")
		fmt.Fprintf(os.Stderr, "  go run tools/get-auth0-token.go -email user@example.com -password secret123\n\n")
		fmt.Fprintf(os.Stderr, "Or set environment variables:\n")
		fmt.Fprintf(os.Stderr, "  export AUTH0_DOMAIN=dev-xxxx.us.auth0.com\n")
		fmt.Fprintf(os.Stderr, "  export AUTH0_CLIENT_ID=your_client_id\n")
		fmt.Fprintf(os.Stderr, "  export AUTH0_AUDIENCE=https://api.howsworkingthere.com\n")
		fmt.Fprintf(os.Stderr, "  export AUTH0_EMAIL=user@example.com\n")
		fmt.Fprintf(os.Stderr, "  export AUTH0_PASSWORD=secret123\n")
		os.Exit(1)
	}

	// Build the token endpoint URL
	tokenURL := fmt.Sprintf("https://%s/oauth/token", *domain)

	// Build request body
	payload := map[string]string{
		"grant_type": *grantType,
		"client_id":  *clientID,
		"username":   *email,
		"password":   *password,
		"audience":   *audience,
		"realm":      *realm,
		"scope":      "openid profile email",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling request: %v\n", err)
		os.Exit(1)
	}

	// Make the request
	resp, err := http.Post(tokenURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error making request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(1)
	}

	// Check for HTTP error
	if resp.StatusCode != http.StatusOK {
		var authErr auth0Error
		if err := json.Unmarshal(body, &authErr); err == nil && authErr.Error != "" {
			fmt.Fprintf(os.Stderr, "Auth0 error: %s - %s\n", authErr.Error, authErr.ErrorDescription)
		} else {
			fmt.Fprintf(os.Stderr, "HTTP error %d: %s\n", resp.StatusCode, string(body))
		}
		os.Exit(1)
	}

	// Parse and output the token response
	var tokenResp auth0TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	// Pretty print the full response
	prettyJSON, _ := json.MarshalIndent(tokenResp, "", "  ")
	fmt.Println(string(prettyJSON))
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
