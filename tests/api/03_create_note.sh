#!/bin/bash
set -e

# Default API URL
API_URL=${API_URL:-"http://localhost:8080"}

echo "=== 3. Creating note for User A ==="

TOKEN_A=$(cat token_a.txt 2>/dev/null || echo "")
if [ -z "$TOKEN_A" ]; then
  echo "Error: token_a.txt not found. Run 02_login.sh first."
  exit 1
fi

echo "Creating note..."
curl -s -X POST -H "Authorization: Bearer $TOKEN_A" -H "Content-Type: application/json" \
  -d '[{"id":"note_a_1","text":"Secret credentials for User A","x":120,"y":340}]' \
  "$API_URL/api/notes"
echo -e "\nNote created for User A.\n"
