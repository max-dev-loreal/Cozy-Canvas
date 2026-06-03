#!/bin/bash
set -e

# Default API URL
API_URL=${API_URL:-"http://localhost:8080"}

echo "=== 4. Granting access and verifying RBAC ==="

TOKEN_B=$(cat token_b.txt 2>/dev/null || echo "")
if [ -z "$TOKEN_B" ]; then
  echo "Error: token_b.txt not found. Run 02_login.sh first."
  exit 1
fi

echo "User B requests access to User A's notes using User A's email, password, and codeword..."
RESPONSE=$(curl -s -X POST -H "Authorization: Bearer $TOKEN_B" -H "Content-Type: application/json" \
  -d '{"email":"user_a@cozy.io","password":"password_a","codeword":"secret_a_1"}' \
  "$API_URL/api/auth/grant-access")

echo "Grant Access Response: $RESPONSE"

OWNER_ID=$(echo "$RESPONSE" | grep -o '"owner_user_id":[0-9]*' | grep -o '[0-9]*')
if [ -z "$OWNER_ID" ]; then
  echo "Error: Failed to obtain owner_user_id from response."
  exit 1
fi

echo -e "\nUser B attempts to fetch User A's notes (owner_id=$OWNER_ID)..."
curl -s -X GET -H "Authorization: Bearer $TOKEN_B" \
  "$API_URL/api/notes?user_id=$OWNER_ID"
echo -e "\n"

echo "User B attempts to modify User A's notes (should be Forbidden)..."
WRITE_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" -X POST -H "Authorization: Bearer $TOKEN_B" -H "Content-Type: application/json" \
  -d '[{"id":"note_a_1","text":"Modified by User B","x":120,"y":340}]' \
  "$API_URL/api/notes?user_id=$OWNER_ID")

echo "Write HTTP Status: $WRITE_RESPONSE (Expected: 403)"
if [ "$WRITE_RESPONSE" -ne 403 ]; then
  echo "Error: Write access was not blocked!"
  exit 1
fi

echo "RBAC and Read-Only access verified successfully!"
