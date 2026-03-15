#!/bin/bash
# get-auth0-token.sh - Shell script to fetch Auth0 access tokens for API testing
#
# Usage:
#   ./tools/get-auth0-token.sh
#   ./tools/get-auth0-token.sh user@example.com secret123
#
# Or with environment variables:
export AUTH0_DOMAIN=dev-ouhyu6xk.auth0.com
export AUTH0_CLIENT_ID=ewpQPkKQMhExYfQdNHFKqdehpvKbDafi
export AUTH0_AUDIENCE=https://dev-ouhyu6xk.auth0.com/api/v2/
#   export AUTH0_EMAIL=user@example.com
#   export AUTH0_PASSWORD=secret123
#   ./tools/get-auth0-token.sh
#
# Requirements: curl, jq (optional, for pretty printing)

set -e

# Get credentials from arguments or environment
EMAIL="${1:-${AUTH0_EMAIL:-}}"
PASSWORD="${2:-${AUTH0_PASSWORD:-}}"

# Required environment variables
DOMAIN="${AUTH0_DOMAIN:-}"
CLIENT_ID="${AUTH0_CLIENT_ID:-}"
AUDIENCE="${AUTH0_AUDIENCE:-}"
REALM="${AUTH0_REALM:-Username-Password-Authentication}"

# Validate inputs
MISSING=""

if [ -z "$DOMAIN" ]; then
    MISSING="${MISSING}\n  - AUTH0_DOMAIN (e.g., dev-xxxx.us.auth0.com)"
fi

if [ -z "$CLIENT_ID" ]; then
    MISSING="${MISSING}\n  - AUTH0_CLIENT_ID (from Auth0 application settings)"
fi

if [ -z "$AUDIENCE" ]; then
    MISSING="${MISSING}\n  - AUTH0_AUDIENCE (e.g., https://api.howsworkingthere.com)"
fi

if [ -z "$EMAIL" ]; then
    MISSING="${MISSING}\n  - AUTH0_EMAIL or first argument (user email)"
fi

if [ -z "$PASSWORD" ]; then
    MISSING="${MISSING}\n  - AUTH0_PASSWORD or second argument (user password)"
fi

if [ -n "$MISSING" ]; then
    echo "Missing required parameters:${MISSING}"
    echo ""
    echo "Usage:"
    echo "  $0 [email] [password]"
    echo ""
    echo "Or set environment variables:"
    echo "  export AUTH0_DOMAIN=dev-xxxx.us.auth0.com"
    echo "  export AUTH0_CLIENT_ID=your_client_id"
    echo "  export AUTH0_AUDIENCE=https://api.howsworkingthere.com"
    echo "  export AUTH0_EMAIL=user@example.com"
    echo "  export AUTH0_PASSWORD=secret123"
    exit 1
fi

# Build the request
echo "Fetching token from Auth0 ($DOMAIN)..."
echo ""

RESPONSE=$(curl -s -X POST "https://${DOMAIN}/oauth/token" \
    -H "Content-Type: application/json" \
    -d "{
        \"grant_type\": \"http://auth0.com/oauth/grant-type/password-realm\",
        \"client_id\": \"${CLIENT_ID}\",
        \"username\": \"${EMAIL}\",
        \"password\": \"${PASSWORD}\",
        \"audience\": \"${AUDIENCE}\",
        \"realm\": \"${REALM}\",
        \"scope\": \"openid profile email\"
    }" 2>/dev/null)

# Check for errors
if echo "$RESPONSE" | grep -q '"error"'; then
    echo "Auth0 error:"
    echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"
    exit 1
fi

# Output the response
if command -v jq &> /dev/null; then
    echo "$RESPONSE" | jq .
else
    echo "$RESPONSE"
fi

echo ""
echo "Copy the access_token value and paste it into Postman's {{auth_token}} variable."
