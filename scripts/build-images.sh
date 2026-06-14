#!/bin/bash
set -e

docker build --progress=plain -f api-gateway/Dockerfile -t  api-gateway:local .

docker build --progress=plain -f services/catalog-read-service/Dockerfile -t catalog-read-service:local .

docker build --progress=plain -f services/catalog-write-service/Dockerfile -t catalog-write-service:local .

docker build --progress=plain -f services/inventory-service/Dockerfile -t inventory-service:local .

docker build --progress=plain -f services/order-service/Dockerfile -t order-service:local .