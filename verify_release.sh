#!/usr/bin/env bash

# Usage: ./verify_release.sh <tag>
# Example: ./verify_release.sh v1.5.15

set -euo pipefail

TAG="${1:-}"
if [ -z "$TAG" ]; then
    echo "Usage: $0 <tag>"
    echo "Example: $0 v1.5.15"
    exit 1
fi

REPO="Mirantis/launchpad"
API_URL="https://api.github.com/repos/$REPO/releases/tags/$TAG"

echo "### Fetching release metadata for $TAG..."
RELEASE_JSON=$(curl -sSf "$API_URL")

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT
cd "$TMP_DIR"

echo "### Processing and verifying assets using digest from API..."
echo "$RELEASE_JSON" | jq -c '.assets[]' | while read -r asset; do
    NAME=$(echo "$asset" | jq -r '.name')
    URL=$(echo "$asset" | jq -r '.browser_download_url')
    DIGEST=$(echo "$asset" | jq -r '.digest')

    # Skip if digest is null or empty
    if [ "$DIGEST" == "null" ] || [ -z "$DIGEST" ]; then
        echo "Skipping $NAME (no digest field found)"
        continue
    fi

    # Extract expected hash from digest (e.g., sha256:abc...)
    EXPECTED_HASH=$(echo "$DIGEST" | cut -d: -f2)

    echo "Downloading $NAME..."
    curl -sSL -o "$NAME" "$URL"

    echo "Verifying $NAME..."
    # Use sha256sum -c - to verify the hash against the file.
    # Note: sha256sum expects TWO spaces between the hash and filename.
    if echo "${EXPECTED_HASH}  ${NAME}" | sha256sum -c -; then
        echo "OK: $NAME (digest matches)"
    else
        echo "FAILED: $NAME (digest mismatch!)"
        exit 1
    fi
    echo ""
done

