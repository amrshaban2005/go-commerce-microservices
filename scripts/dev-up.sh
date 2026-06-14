#!/bin/bash
set -e

docker compose --env-file deployments/.env.infrastructure -f deployments/docker-compose.infrastructure.yml up -d