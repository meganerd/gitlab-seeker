#!/bin/bash
# Build GitLab Scanner for all major platforms

VERSION=${1:-dev}
OUTPUT_DIR="dist"

echo "Building GitLab Scanner v$VERSION for all platforms..."
echo ""

mkdir -p "$OUTPUT_DIR"

# Linux
echo "→ Building Linux AMD64..."
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.Version=$VERSION" -o "$OUTPUT_DIR/scanner-linux-amd64" ./cmd/scanner

echo "→ Building Linux ARM64..."
GOOS=linux GOARCH=arm64 go build -ldflags "-s -w -X main.Version=$VERSION" -o "$OUTPUT_DIR/scanner-linux-arm64" ./cmd/scanner

# macOS
echo "→ Building macOS AMD64 (Intel)..."
GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -X main.Version=$VERSION" -o "$OUTPUT_DIR/scanner-darwin-amd64" ./cmd/scanner

echo "→ Building macOS ARM64 (Apple Silicon)..."
GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w -X main.Version=$VERSION" -o "$OUTPUT_DIR/scanner-darwin-arm64" ./cmd/scanner

# Windows
echo "→ Building Windows AMD64..."
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -X main.Version=$VERSION" -o "$OUTPUT_DIR/scanner-windows-amd64.exe" ./cmd/scanner

echo "→ Building Windows ARM64..."
GOOS=windows GOARCH=arm64 go build -ldflags "-s -w -X main.Version=$VERSION" -o "$OUTPUT_DIR/scanner-windows-arm64.exe" ./cmd/scanner

echo ""
echo "✓ Builds complete in $OUTPUT_DIR/"
echo ""
ls -lh "$OUTPUT_DIR/"

echo ""
echo "Creating checksums..."
cd "$OUTPUT_DIR"
sha256sum scanner-* > checksums.txt
echo "✓ Checksums saved to $OUTPUT_DIR/checksums.txt"
