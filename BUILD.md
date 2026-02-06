# Building GitLab Project Scanner

## Quick Build (Current Platform)

```bash
# Build for your current OS and architecture
go build -o scanner ./cmd/scanner

# Run it
./scanner --help
```

## Cross-Platform Builds

### Prerequisites
- Go 1.21 or later
- Git (for version info)

### Build for Specific Platforms

Go supports cross-compilation. Set `GOOS` and `GOARCH` environment variables:

#### Linux AMD64 (x86_64)
```bash
GOOS=linux GOARCH=amd64 go build -o scanner-linux-amd64 ./cmd/scanner
```

#### Linux ARM64 (Raspberry Pi, AWS Graviton)
```bash
GOOS=linux GOARCH=arm64 go build -o scanner-linux-arm64 ./cmd/scanner
```

#### macOS Intel (AMD64)
```bash
GOOS=darwin GOARCH=amd64 go build -o scanner-darwin-amd64 ./cmd/scanner
```

#### macOS Apple Silicon (ARM64)
```bash
GOOS=darwin GOARCH=arm64 go build -o scanner-darwin-arm64 ./cmd/scanner
```

#### Windows AMD64
```bash
GOOS=windows GOARCH=amd64 go build -o scanner-windows-amd64.exe ./cmd/scanner
```

#### Windows ARM64 (Surface Pro X)
```bash
GOOS=windows GOARCH=arm64 go build -o scanner-windows-arm64.exe ./cmd/scanner
```

### Build All Platforms at Once

Use the included build script:

```bash
#!/bin/bash
# build-all.sh - Build for all major platforms

VERSION=${1:-dev}
OUTPUT_DIR="dist"

mkdir -p "$OUTPUT_DIR"

# Linux
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$VERSION" -o "$OUTPUT_DIR/scanner-linux-amd64" ./cmd/scanner
GOOS=linux GOARCH=arm64 go build -ldflags "-X main.Version=$VERSION" -o "$OUTPUT_DIR/scanner-linux-arm64" ./cmd/scanner

# macOS
GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=$VERSION" -o "$OUTPUT_DIR/scanner-darwin-amd64" ./cmd/scanner
GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.Version=$VERSION" -o "$OUTPUT_DIR/scanner-darwin-arm64" ./cmd/scanner

# Windows
GOOS=windows GOARCH=amd64 go build -ldflags "-X main.Version=$VERSION" -o "$OUTPUT_DIR/scanner-windows-amd64.exe" ./cmd/scanner
GOOS=windows GOARCH=arm64 go build -ldflags "-X main.Version=$VERSION" -o "$OUTPUT_DIR/scanner-windows-arm64.exe" ./cmd/scanner

echo "Builds complete in $OUTPUT_DIR/"
ls -lh "$OUTPUT_DIR/"
```

Run it:
```bash
chmod +x build-all.sh
./build-all.sh v1.0.0
```

### Optimized Production Builds

For smaller binaries and better performance:

```bash
# Strip debug info and disable symbol table
go build -ldflags="-s -w" -o scanner ./cmd/scanner

# For specific platform (example: Linux AMD64)
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o scanner-linux-amd64 ./cmd/scanner
```

Size comparison:
- Normal build: ~15-20 MB
- Optimized build: ~10-12 MB

### Compression

Further reduce binary size with UPX (optional):

```bash
# Install UPX first (macOS: brew install upx, Linux: apt install upx-ucl)
upx --best scanner

# Can reduce size by 60-70%
# Normal: 15 MB → Compressed: 5-6 MB
```

**Note:** Some antivirus software flags UPX-compressed binaries as suspicious.

## Platform-Specific Notes

### macOS
On macOS, the binary may be quarantined on first run:
```bash
# Remove quarantine attribute
xattr -d com.apple.quarantine scanner-darwin-arm64

# Or sign the binary (requires Apple Developer account)
codesign -s "Developer ID" scanner-darwin-arm64
```

### Linux
Make the binary executable:
```bash
chmod +x scanner-linux-amd64
```

### Windows
No special steps needed. Run from PowerShell or Command Prompt:
```powershell
.\scanner-windows-amd64.exe --help
```

## Supported Platforms

### Tier 1 (Fully Tested)
- Linux AMD64 (x86_64)
- macOS ARM64 (Apple Silicon)
- macOS AMD64 (Intel)

### Tier 2 (Should Work)
- Linux ARM64 (aarch64)
- Windows AMD64 (x86_64)
- Windows ARM64

### Other Platforms

Go supports many more platforms. See full list:
```bash
go tool dist list
```

Build for any supported platform:
```bash
GOOS=<os> GOARCH=<arch> go build -o scanner ./cmd/scanner
```

## Docker Build

Create a multi-platform Docker image:

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY . .
RUN go build -ldflags="-s -w" -o scanner ./cmd/scanner

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /build/scanner .
ENTRYPOINT ["/app/scanner"]
```

Build:
```bash
# Single platform
docker build -t gitlab-scanner .

# Multi-platform (requires buildx)
docker buildx build --platform linux/amd64,linux/arm64 -t gitlab-scanner .
```

## Release Process

For GitHub/GitLab releases:

```bash
# Tag the release
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# Build all platforms
./build-all.sh v1.0.0

# Create checksums
cd dist/
sha256sum scanner-* > checksums.txt

# Upload to releases page
```

## Troubleshooting

### Build Fails with "package not found"
```bash
# Update dependencies
go mod tidy
go mod download
```

### Cross-compilation Issues
```bash
# Some packages may not support cross-compilation
# Check if any CGO dependencies exist
go list -f '{{if .CgoFiles}}{{.ImportPath}}{{end}}' ./...

# If CGO is needed, install cross-compilation toolchain
# or build in a VM/container for that platform
```

### Binary Too Large
```bash
# Use build flags to reduce size
go build -ldflags="-s -w" -trimpath -o scanner ./cmd/scanner

# Optionally compress with UPX
upx --best scanner
```

## Verification

After building, verify the binary:

```bash
# Check version/help
./scanner --help

# Run tests on target platform
go test ./...

# Test basic functionality
./scanner --url https://gitlab.com/test --token dummy --help
```

## Quick Reference

| Platform | Command |
|----------|---------|
| **Linux x64** | `GOOS=linux GOARCH=amd64 go build -o scanner-linux-amd64 ./cmd/scanner` |
| **Linux ARM64** | `GOOS=linux GOARCH=arm64 go build -o scanner-linux-arm64 ./cmd/scanner` |
| **macOS Intel** | `GOOS=darwin GOARCH=amd64 go build -o scanner-darwin-amd64 ./cmd/scanner` |
| **macOS M1/M2/M3** | `GOOS=darwin GOARCH=arm64 go build -o scanner-darwin-arm64 ./cmd/scanner` |
| **Windows x64** | `GOOS=windows GOARCH=amd64 go build -o scanner.exe ./cmd/scanner` |
| **Windows ARM** | `GOOS=windows GOARCH=arm64 go build -o scanner-arm64.exe ./cmd/scanner` |
