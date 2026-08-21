#!/usr/bin/env bash
set -e

OUTPUT_DIR="${1:-bin}"
mkdir -p "$OUTPUT_DIR"
TARGET="$OUTPUT_DIR/agpass"

echo "Building agent-pass -> $TARGET..."
go build -ldflags="-s -w" -o "$TARGET" .
echo "Build successful! Output: $TARGET"