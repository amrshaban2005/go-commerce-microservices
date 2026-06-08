#!/bin/bash
set -e

PROJECT_ROOT=$(pwd)

echo "Installing project tools into ./bin..."

GOBIN="$PROJECT_ROOT/bin" go install github.com/pressly/goose/v3/cmd/goose@latest
GOBIN="$PROJECT_ROOT/bin" go install github.com/air-verse/air@latest
GOBIN="$PROJECT_ROOT/bin" go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
GOBIN="$PROJECT_ROOT/bin" go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

echo "Tools installed successfully."