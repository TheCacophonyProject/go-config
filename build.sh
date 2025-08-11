#!/bin/bash
# Set GOOS and GOARCH to force ARM64 builds
export GOOS=linux
export GOARCH=arm64

# Build snapshot
goreleaser release --snapshot --clean --skip=publish

# Show the created deb files
echo "Created deb packages:"
find dist -name "*.deb" -type f
