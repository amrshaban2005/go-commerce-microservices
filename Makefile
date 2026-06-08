install-tools:
	./scripts/install-tools.sh

dev-up:
	./scripts/dev-up.sh

dev-down:
	./scripts/dev-down.sh

dev-reset:
	./scripts/dev-reset.sh

migrate-up:
	./scripts/migrate-up.sh

migrate-down:
	./scripts/migrate-down.sh

proto:
	./scripts/generate-proto.sh

run-order:
	cd services/order-service && ../../bin/air

run-inventory:
	cd services/inventory-service && ../../bin/air

run-catalog-write:
	cd services/catalog-write-service && ../../bin/air

run-catalog-read:
	cd services/catalog-read-service && ../../bin/air

run-notification:
	cd services/notification-service && ../../bin/air

run-api-gateway:
	cd api-gateway && ../bin/air

dev-start:
	make dev-up
	sleep 5
	make migrate-up

dev-stop:
	make dev-down
	make migrate-down
