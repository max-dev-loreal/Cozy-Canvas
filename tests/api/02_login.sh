#!/bin/bash
set -e

# Default API URL
API_URL=${API_URL:-"http://localhost:8080"}

echo "=== 2. Logging in test users ==="

# Login User A
echo "Logging in User A..."
RESPONSE_A=$(curl -s -X POST -H "Content-Type: application/json" \
  -d '{"email":"user_a@cozy.io","password":"password_a"}' \
  "$API_URL/api/auth/login")

TOKEN_A=$(echo "$RESPONSE_A" | grep -o '"token":"[^"]*' | grep -o '[^"]*$')
if [ -z "$TOKEN_A" ]; then
  echo "Error: Failed to login User A"
  echo "Response: $RESPONSE_A"
  exit 1
fi
echo "$TOKEN_A" > token_a.txt
echo "User A token saved."

# Login User B
echo "Logging in User B..."
RESPONSE_B=$(curl -s -X POST -H "Content-Type: application/json" \
  -d '{"email":"user_b@cozy.io","password":"password_b"}' \
  "$API_URL/api/auth/login")

TOKEN_B=$(echo "$RESPONSE_B" | grep -o '"token":"[^"]*' | grep -o '[^"]*$')
if [ -z "$TOKEN_B" ]; then
  echo "Error: Failed to login User B"
  echo "Response: $RESPONSE_B"
  exit 1
fi
echo "$TOKEN_B" > token_b.txt
echo "User B token saved."
echo ""
