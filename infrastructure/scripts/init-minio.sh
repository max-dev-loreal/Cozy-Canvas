#!/bin/sh

# Cozy Canvas - MinIO Bucket Auto-Initialization Script
# This script runs inside a client container to configure objects storage.

echo "Waiting for MinIO object storage service to become ready..."
until curl -s http://minio:9000/minio/health/live > /dev/null 2>&1; do
  sleep 1
done

echo "[MinIO] Storage is up! Logging in..."

# Setup access credentials in mc client CLI
mc alias set cozyminio http://minio:9000 ${MINIO_ROOT_USER:-cozyadmin} ${MINIO_ROOT_PASSWORD:-cozysecret}

BUCKET_NAME=${MINIO_BUCKET_NAME:-cozy-canvas-assets}

# Create bucket if it doesn't already exist
if ! mc ls cozyminio/${BUCKET_NAME} > /dev/null 2>&1; then
  echo "[MinIO] Creating public assets bucket: '${BUCKET_NAME}'..."
  mc mb cozyminio/${BUCKET_NAME}
  # Set anonymous read permissions (ideal for sharing image uploads in notes)
  mc anonymous set download cozyminio/${BUCKET_NAME}
  echo "[MinIO] Bucket '${BUCKET_NAME}' successfully initialized and made public."
else
  echo "[MinIO] Bucket '${BUCKET_NAME}' already exists, skipping creation."
fi
