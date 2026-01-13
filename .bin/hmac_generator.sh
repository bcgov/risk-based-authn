#!/usr/bin/env bash

# Match values from your .env
KEY_ID="abcd1234"
SECRET="supersecret1"

# Current Unix timestamp (seconds)
TIMESTAMP=$(date +%s)

# Message to sign (timestamp only, same as Go code)
MESSAGE="$TIMESTAMP"

# Generate HMAC-SHA256 signature and hex-encode it
SIGNATURE=$(printf '%s' "$MESSAGE" \
  | openssl dgst -sha256 -hmac "$SECRET" \
  | awk '{print $NF}')

# Output headers
echo "X-Key-ID: $KEY_ID"
echo "X-Timestamp: $TIMESTAMP"
echo "X-Signature: $SIGNATURE"
