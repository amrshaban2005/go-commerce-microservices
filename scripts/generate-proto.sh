#!/bin/bash
set -e

protoc \
  -I api/proto \
  --go_out=api/gen/go \
  --go-grpc_out=api/gen/go \
  --go_opt=paths=source_relative \
  --go-grpc_opt=paths=source_relative \
  catalog/v1/catalog.proto\
  order/v1/order.proto

echo "Proto files generated successfully."