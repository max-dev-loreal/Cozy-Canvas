#!/bin/bash
set -e

# Default API URL
API_URL=${API_URL:-"http://localhost:8080"}

echo "=== 5. Creating connections and testing file uploads ==="

TOKEN_A=$(cat token_a.txt 2>/dev/null || echo "")
if [ -z "$TOKEN_A" ]; then
  echo "Error: token_a.txt not found. Run 02_login.sh first."
  exit 1
fi

# 1. Create notes and connection atomically using /api/sync
echo "Synchronizing graph (notes and connection)..."
curl -s -X POST -H "Authorization: Bearer $TOKEN_A" -H "Content-Type: application/json" \
  -d '{"notes":[{"id":"note_a_1","text":"Note A1","x":100,"y":100},{"id":"note_a_2","text":"Note A2","x":200,"y":200}],"connections":[{"id":"note_a_1-note_a_2","source":"note_a_1","target":"note_a_2"}]}' \
  "$API_URL/api/sync"
echo -e "\nGraph synchronized successfully."

# 3. Request presigned upload URL
echo -e "\nRequesting presigned upload URL..."
UPLOAD_RESPONSE=$(curl -s -X POST -H "Authorization: Bearer $TOKEN_A" -H "Content-Type: application/json" \
  -d '{"filename":"test.txt"}' \
  "$API_URL/api/files/upload-url")
echo "Upload URL Response: $UPLOAD_RESPONSE"

UPLOAD_URL=$(echo "$UPLOAD_RESPONSE" | grep -o '"uploadUrl":"[^"]*' | grep -o '[^"]*$')
FILE_ID=$(echo "$UPLOAD_RESPONSE" | grep -o '"filename":"[^"]*' | grep -o '[^"]*$')

if [ -z "$UPLOAD_URL" ] || [ -z "$FILE_ID" ]; then
  echo "Error: Failed to obtain uploadUrl or filename from response."
  exit 1
fi

# 4. Upload file directly to S3/MinIO
echo -e "\nUploading file content directly to MinIO..."
echo "This is a test file for Cozy Canvas E2E testing." > temp_test.txt
curl -s -X PUT -T temp_test.txt -H "Content-Type: text/plain" "$UPLOAD_URL"
rm -f temp_test.txt
echo "File uploaded successfully."

# 5. Get presigned download URL
echo -e "\nRetrieving download URL..."
DOWNLOAD_RESPONSE=$(curl -s -X GET -H "Authorization: Bearer $TOKEN_A" \
  "$API_URL/api/files/download-url/$FILE_ID")
echo "Download URL Response: $DOWNLOAD_RESPONSE"

DOWNLOAD_URL=$(echo "$DOWNLOAD_RESPONSE" | grep -o '"downloadUrl":"[^"]*' | grep -o '[^"]*$')
if [ -z "$DOWNLOAD_URL" ]; then
  echo "Error: Failed to obtain download URL."
  exit 1
fi

echo -e "\nAccessing the uploaded file via presigned GET URL..."
FILE_CONTENT=$(curl -s "$DOWNLOAD_URL")
echo "File Content: '$FILE_CONTENT'"

if [ "$FILE_CONTENT" != "This is a test file for Cozy Canvas E2E testing." ]; then
  echo "Error: File content does not match!"
  exit 1
fi

echo "File upload and download verified successfully!"
