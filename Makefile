MODULES := \
	api-gateway \
	api/gen/go \
	pkg \
	services/catalog-read-service \
	services/catalog-write-service \
	services/inventory-service \
	services/order-service

install-tools:
	./scripts/install-tools.sh

dev-up:
	./scripts/dev-up.sh

dev-down:
	./scripts/dev-down.sh

deploy-up:
	./scripts/deploy-up.sh

deploy-down:
	./scripts/deploy-down.sh

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

run-docker-build:
	./scripts/build-images.sh

run-check-health:
	./scripts/check-health.sh

fmt:
	gofmt -w $$(git ls-files '*.go')

test:
	@for module in $(MODULES); do \
		echo "Testing $$module"; \
		(cd $$module && go test ./...) || exit 1; \
	done

vet:
	@for module in $(MODULES); do \
		echo "Vetting $$module"; \
		(cd $$module && go vet ./...) || exit 1; \
	done

dev-check:
	make fmt
	make test
	make vet

dev-start:
	make dev-up
	sleep 5
	make migrate-up

dev-stop:
	make dev-down

prod-start:
	make run-docker-build
	make deploy-up
	sleep 5
	make migrate-up

prod-stop:
	make deploy-down


