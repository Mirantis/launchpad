#!/usr/bin/env bash

# Usage: ./verify_release.sh <tag>
# Example: ./verify_release.sh v1.5.19
#
# Verifies published release assets against checksums.sha256, the checksum
# record produced at build time by .github/workflows/release.yml with
# `sha256sum launchpad_* > checksums.sha256` and published with the release.
#
# The release API's `.digest` field is deliberately NOT used. GitHub computes
# it from the bytes it currently stores, so comparing a download against it
# only proves the download matches what GitHub holds: it detects transfer
# corruption and cannot detect an asset replaced at rest.
#
# What this does and does not prove: checksums.sha256 is itself an unsigned
# release asset, so an attacker able to replace a binary could replace the
# checksum record too. This is build-time provenance, not cryptographic
# authentication. Authenticating a release requires signed artifacts or build
# attestations.
#
# Exit status is 0 only when at least one asset was verified and every asset
# verified successfully. "Nothing could be verified" is a failure, not a pass.
#
# Releases published before the move to GitHub Actions carry per-asset
# <name>.sha256 sidecars instead of checksums.sha256 and are not supported;
# the tool reports what it found and exits non-zero rather than passing.

set -euo pipefail

TAG="${1:-}"
if [ -z "$TAG" ]; then
    echo "Usage: $0 <tag>"
    echo "Example: $0 v1.5.19"
    exit 1
fi

REPO="Mirantis/launchpad"
API_URL="https://api.github.com/repos/$REPO/releases/tags/$TAG"

echo "### Fetching release metadata for $TAG..."
RELEASE_JSON=$(curl -sSf "$API_URL")

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT
cd "$TMP_DIR"

# Download URL for a named asset, empty if the release has no such asset.
asset_url() {
    echo "$RELEASE_JSON" | jq -r --arg name "$1" \
        '.assets[] | select(.name == $name) | .browser_download_url'
}

# Binary assets are everything that is not itself a checksum record. This also
# excludes checksums.sha256, which ends in .sha256.
mapfile -t ASSETS < <(echo "$RELEASE_JSON" | jq -r '.assets[].name' | grep -v '\.sha256$' || true)

if [ ${#ASSETS[@]} -eq 0 ]; then
    echo "FAILED: release $TAG publishes no verifiable assets"
    exit 1
fi

CHECKSUMS_URL=$(asset_url "checksums.sha256")
if [ -z "$CHECKSUMS_URL" ]; then
    # Say what the release actually contains, so the reader learns why it
    # cannot be verified instead of being left with an opaque error.
    sidecars=$(echo "$RELEASE_JSON" | jq '[.assets[].name | select(endswith(".sha256"))] | length')
    echo "FAILED: release $TAG publishes no checksums.sha256, so there is no"
    echo "        build-time checksum record to verify its assets against."
    echo "        Found ${#ASSETS[@]} binary asset(s) and ${sidecars} .sha256 file(s)."
    if [ "$sidecars" -gt 0 ]; then
        echo "        Releases published before the move to GitHub Actions carry one"
        echo "        <name>.sha256 sidecar per binary rather than a single"
        echo "        checksums.sha256. That layout is not supported by this tool."
    fi
    echo "        Nothing was verified. Exiting non-zero rather than reporting success."
    exit 1
fi

echo "### Checksum source: checksums.sha256"
curl -sSfL -o checksums.sha256 "$CHECKSUMS_URL"

# The expected hash for an asset, from the build-time record. Fails if the
# record has no usable entry for it.
expected_hash() {
    local want="$1" hash
    hash=$(awk -v want="$want" '$2 == want { print $1 }' checksums.sha256)
    if [[ ! $hash =~ ^[0-9a-fA-F]{64}$ ]]; then
        return 1
    fi
    echo "$hash" | tr '[:upper:]' '[:lower:]'
}

echo "### Verifying ${#ASSETS[@]} asset(s) for $TAG..."
verified=0
failed=0

# A plain loop, not a pipeline into `while`: the counters must survive the
# iteration, which they would not do in the subshell a pipeline creates.
for name in "${ASSETS[@]}"; do
    if ! expected=$(expected_hash "$name"); then
        echo "FAILED: $name (no entry in checksums.sha256)"
        failed=$((failed + 1))
        continue
    fi

    # A download or digest error is this asset's failure, not grounds to
    # abandon the run: under `set -e` an unguarded failure here would skip
    # every remaining asset and the summary with them.
    if ! curl -sSfL -o "$name" "$(asset_url "$name")"; then
        echo "FAILED: $name (download failed)"
        failed=$((failed + 1))
        continue
    fi

    if ! actual=$(sha256sum "$name" | cut -d' ' -f1); then
        echo "FAILED: $name (could not compute checksum of the download)"
        failed=$((failed + 1))
        continue
    fi

    if [ "$actual" = "$expected" ]; then
        echo "OK: $name"
        verified=$((verified + 1))
    else
        echo "FAILED: $name (expected $expected, got $actual)"
        failed=$((failed + 1))
    fi
done

echo "### $verified verified, $failed failed, of ${#ASSETS[@]} asset(s)"

if [ "$failed" -gt 0 ]; then
    exit 1
fi

# Every asset was skipped: report failure rather than a pass over nothing.
if [ "$verified" -eq 0 ]; then
    echo "FAILED: no assets were verified for $TAG"
    exit 1
fi

echo "### $TAG verified against its build-time checksums"
