#!/bin/bash
set -e

# Default API URL
API_URL=${API_URL:-"http://localhost:8080"}

echo "=== 1. Registering test users ==="

# Register User A (owner)
echo "Registering User A (owner)..."
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"username":"user_a","email":"user_a@cozy.io","password":"password_a","codewords":["secret_a_1","secret_a_2"]}' \
  "$API_URL/api/auth/register"
echo -e "\n"

# Register User B (viewer)
echo "Registering User B (viewer)..."
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"username":"user_b","email":"user_b@cozy.io","password":"password_b","codewords":["secret_b_1","secret_b_2"]}' \
  "$API_URL/api/auth/register"
echo -e "\n"
