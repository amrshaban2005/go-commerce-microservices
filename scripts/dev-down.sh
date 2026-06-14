#!/bin/bash
set -e

docker compose --env-file deployments/.env.infrastructure -f deployments/docker-compose.infrastructure.yml down

docker compose --env-file deployments/.env.services -f deployments/docker-compose.services.yml down