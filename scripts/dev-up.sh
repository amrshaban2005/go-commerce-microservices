#!/bin/bash
set -e

docker compose --env-file deployments/.env.infrastructure -f deployments/docker-compose.infrastructure.yml up -d

docker compose --env-file deployments/.env.services -f deployments/docker-compose.services.yml up -d 