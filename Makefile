SHELL := /bin/bash

GO_MODULES := \
	api-gateway \
	api/gen/go \
	pkg \
	services/catalog-read-service \
	services/catalog-write-service \
	services/inventory-service \
	services/order-service

APP_MODULES := \
	api-gateway \
	services/catalog-read-service \
	services/catalog-write-service \
	services/inventory-service \
	services/order-service

.PHONY: fmt fmt-check vet lint test test-ci build

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

swagger:
	cd api-gateway && swag init \
		-g cmd/main.go \
		-o docs \
		--parseInternal

fmt:
	gofmt -w $$(git ls-files '*.go')

fmt-check:
	@unformatted="$$(gofmt -l $$(git ls-files '*.go'))"; \
	if [ -n "$$unformatted" ]; then \
		echo "The following files are not formatted:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

test:
	@for module in $(GO_MODULES); do \
		echo "Testing $$module"; \
		(cd $$module && go test ./... -count=1) || exit 1; \
	done

test-ci:
	@rm -rf artifacts/test-results
	@mkdir -p artifacts/test-results
	@set -o pipefail; status=0; \
	for module in $(GO_MODULES); do \
		echo "Testing $$module"; \
		report="$$(echo "$$module" | tr '/' '-').json"; \
		(cd "$$module" && go test ./... -count=1 -json) \
			| tee "artifacts/test-results/$$report" || status=1; \
	done; \
	exit $$status

test-e2e:
	cd services/test/e2e && gotestsum ./... -count=1 -v

vet:
	@for module in $(GO_MODULES); do \
		echo "Vetting $$module"; \
		(cd "$$module" && go vet ./...) || exit 1; \
	done

lint:
	@for module in $(GO_MODULES); do \
		echo "Linting $$module"; \
		(cd "$$module" && golangci-lint run --config "$(CURDIR)/.golangci.yml" ./...) || exit 1; \
	done

build:
	@for module in $(APP_MODULES); do \
		echo "Building $$module"; \
		(cd "$$module" && go build ./...) || exit 1; \
	done

generate-mock:
	./scripts/generate-mocks.sh

dev-check:
	make fmt
	make test
	make vet
	make test-e2e

dev-start:
	make dev-up
	make migrate-up

dev-stop:
	make dev-down

prod-start:
	make run-docker-build
	make dev-up
	make migrate-up
	make deploy-up

prod-stop:
	make deploy-down
