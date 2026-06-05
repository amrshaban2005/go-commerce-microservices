#!/bin/bash
set -e

docker compose --env-file .env -f deployments/docker-compose.yml down -v
docker compose --env-file .env -f deployments/docker-compose.yml up -d