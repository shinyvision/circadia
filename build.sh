#!/bin/bash
set -e

echo "Starting build script..."

# Clear potential conflicting flags
export CFLAGS=""
export CXXFLAGS=""
export LDFLAGS=""
export GOFLAGS=""

echo "Environment flags cleared."

# Standard Go path in Freedesktop SDK
GO=/usr/lib/sdk/golang/bin/go

echo "Using Go: $($GO version)"

# Build the binary
# -mod=vendor: use local vendor directory
# -v: verbose
# -o circadia-bin: output file
echo "Running build..."
$GO build -mod=vendor -v -o circadia-bin .

echo "Build complete. Output:"
ls -l circadia-bin
