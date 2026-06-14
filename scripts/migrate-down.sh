#!/bin/bash
set -e

set -a
source deployments/.env.infrastructure
set +a

./bin/goose -dir services/catalog-write-service/migrations postgres \
"postgres://${CATALOG_WRITE_DB_USER}:${CATALOG_WRITE_DB_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${CATALOG_WRITE_DB_NAME}?sslmode=disable" down

./bin/goose -dir services/inventory-service/migrations postgres \
"postgres://${INVENTORY_DB_USER}:${INVENTORY_DB_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${INVENTORY_DB_NAME}?sslmode=disable" down

./bin/goose -dir services/order-service/migrations postgres \
"postgres://${ORDER_DB_USER}:${ORDER_DB_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${ORDER_DB_NAME}?sslmode=disable" down

./bin/goose -dir services/notification-service/migrations postgres \
"postgres://${NOTIFICATION_DB_USER}:${NOTIFICATION_DB_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${NOTIFICATION_DB_NAME}?sslmode=disable" down