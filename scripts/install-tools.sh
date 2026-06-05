#!/bin/bash
set -e

PROJECT_ROOT=$(pwd)

echo "Installing project tools into ./bin..."

GOBIN="$PROJECT_ROOT/bin" go install github.com/pressly/goose/v3/cmd/goose@latest
GOBIN="$PROJECT_ROOT/bin" go install github.com/air-verse/air@latest

echo "Tools installed successfully."

echo "Goose version:"
./bin/goose -version

echo "Air version:"
./bin/air -v