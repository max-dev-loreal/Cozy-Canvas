#!/bin/bash
set -e

# Change directory to script location
cd "$(dirname "$0")"

# Clean up any leftover token files
rm -f token_a.txt token_b.txt

echo "============================================="
echo "Running Cozy Canvas API Integration Tests"
echo "============================================="

./01_register.sh
./02_login.sh
./03_create_note.sh
./04_grant_access.sh
./05_file_and_connection.sh

# Clean up token files
rm -f token_a.txt token_b.txt

echo "============================================="
echo "All Cozy Canvas API Integration Tests Passed!"
echo "============================================="
