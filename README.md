# Go Commerce Microservices

`Go Commerce Microservices` is a learning-focused ecommerce backend built with Go and a small set of practical microservices patterns: gRPC service communication, REST API gateway, event-driven messaging with RabbitMQ, Saga workflow, Postgres write models, MongoDB read models, Outbox/Inbox patterns, dependency injection with Uber Fx, structured logging with Zap, and configuration with Viper.

## Technologies

- Go
- gRPC
- Gin
- RabbitMQ
- Postgres
- MongoDB
- GORM
- Uber Fx
- Zap
- Viper
- godotenv
- go-mediatr
- go-ozzo/ozzo-validation
- go-playground/validator
- Testcontainers
- Docker Compose
- GitHub Actions

## Features

- Using `Microservices Architecture` with separate services for catalog, orders, inventory, and API gateway.
- Using `Hexagonal Architecture` style inside services with `domain`, `port`, `service`, and `adapter` packages.
- Using `gRPC` for internal service APIs.
- Using a REST `API Gateway` with Gin.
- Using `Event Driven Architecture` with RabbitMQ.
- Using the `Outbox Pattern` for reliable message publishing from write-side services.
- Using the `Inbox Pattern` for idempotent message consumption.
- Using the `Saga Pattern` for the order and stock reservation workflow.
- Using `CQRS` between catalog write and catalog read services.
- Using `Mediator Pattern` in catalog read service with `go-mediatr`.
- Using `Postgres` for write-side services.
- Using `MongoDB` for catalog read projections.
- Using `GORM` for Postgres persistence.
- Using `Uber Fx` for dependency injection and application lifecycle.
- Using `Zap` for structured logging.
- Using `Viper` and `.env` files for configuration.
- Using `go-ozzo/ozzo-validation` for application input validation.
- Using `go-playground/validator` for transport/request validation where needed.
- Using Docker and Docker Compose for local infrastructure and service deployment.
- Using unit tests, integration tests, and E2E tests for selected flows.
- Using GitHub Actions CI for formatting, vet, tests, and build.

## Services

### API Gateway

Path:

```text
api-gateway/
```

The API Gateway exposes HTTP endpoints and calls backend services through gRPC.

Main responsibilities:

- Receive external HTTP requests.
- Validate and map request DTOs.
- Call catalog/order gRPC services.
- Return HTTP responses to clients.

### Catalog Write Service

Path:

```text
services/catalog-write-service/
```

The write side of catalog. It stores product data in Postgres and publishes product events through the outbox.

Main responsibilities:

- Create/update catalog products.
- Store product write data in Postgres.
- Save product integration events in the outbox.
- Publish outbox messages to RabbitMQ.

### Catalog Read Service

Path:

```text
services/catalog-read-service/
```

The read side of catalog. It consumes product events and stores read projections in MongoDB.

Main responsibilities:

- Consume catalog events from RabbitMQ.
- Project product data into MongoDB.
- Serve product read queries over gRPC.
- Use CQRS/Mediator handlers for catalog read features.

### Order Service

Path:

```text
services/order-service/
```

Handles order creation and order status changes.

Main responsibilities:

- Create orders in Postgres.
- Save `ReserveStockRequested` events in the outbox.
- Publish order outbox messages to RabbitMQ.
- Consume stock result events.
- Confirm or reject orders based on inventory results.

### Inventory Service

Path:

```text
services/inventory-service/
```

Handles stock reservation.

Main responsibilities:

- Consume `ReserveStockRequested` events.
- Lock inventory rows during reservation using Postgres transactions.
- Reserve stock when quantity is available.
- Publish `StockReserved` or `StockReservationFailed` events through the outbox.

### Notification Service

Path:

```text
services/notification-service/
```

This service exists in the repository structure, but it is not part of the main running workflow yet.

## System Architecture

```mermaid
flowchart LR
    Client["Client / Browser"]
    Gateway["API Gateway<br/>Gin HTTP API"]

    CatalogWrite["Catalog Write Service<br/>gRPC + Postgres"]
    CatalogRead["Catalog Read Service<br/>gRPC + MongoDB"]
    Order["Order Service<br/>gRPC + Postgres"]
    Inventory["Inventory Service<br/>RabbitMQ Consumer + Postgres"]

    CatalogWriteDB[("Catalog Write DB<br/>Postgres")]
    OrderDB[("Order DB<br/>Postgres")]
    InventoryDB[("Inventory DB<br/>Postgres")]
    CatalogReadDB[("Catalog Read DB<br/>MongoDB")]

    Rabbit["RabbitMQ<br/>Message Broker"]

    Client --> Gateway
    Gateway -->|"HTTP -> gRPC"| CatalogWrite
    Gateway -->|"HTTP -> gRPC"| CatalogRead
    Gateway -->|"HTTP -> gRPC"| Order

    CatalogWrite --> CatalogWriteDB
    Order --> OrderDB
    Inventory --> InventoryDB
    CatalogRead --> CatalogReadDB

    CatalogWrite -->|"ProductCreated / ProductUpdated / ProductDeleted"| Rabbit
    Order -->|"ReserveStockRequested"| Rabbit
    Inventory -->|"StockReserved / StockReservationFailed"| Rabbit

    Rabbit --> CatalogRead
    Rabbit --> Inventory
    Rabbit --> Order
```

The project uses synchronous communication for request/response APIs and asynchronous messaging for cross-service workflows.

- Synchronous path: API Gateway calls backend services through gRPC.
- Asynchronous path: services publish integration events through RabbitMQ.
- Write services use Postgres for transactional data.
- Catalog read service uses MongoDB as a read projection.

## Code Architecture

Most services use a hexagonal architecture style. The application core owns the domain rules and ports, while adapters handle gRPC, RabbitMQ, and database details.

```mermaid
flowchart TB
    External["External world<br/>gRPC / RabbitMQ / Database"]

    subgraph Adapter["adapter"]
        GRPC["gRPC adapter"]
        Messaging["Messaging adapter"]
        Repository["Repository adapter"]
    end

    subgraph Core["core"]
        Port["port interfaces"]
        Service["application service"]
        Domain["domain model"]
    end

    DB[("Database")]
    Broker["RabbitMQ"]

    GRPC --> Service
    Messaging --> Service
    Service --> Domain
    Service --> Port
    Port --> Repository
    Repository --> DB
    Messaging --> Broker
    External --> GRPC
    External --> Messaging
```

Catalog read service also practices a feature-based CQRS/Mediator shape. That flow is:

```text
adapter -> query/command -> mediator -> handler -> repository port -> repository adapter -> database
```

## Saga Pattern

The order workflow uses an event-driven Saga. There is no single distributed database transaction across order and inventory services. Instead, each service commits its own local transaction and publishes the next event.

```mermaid
flowchart LR
    CreateOrder["Create Order<br/>Order Service"]
    ReserveRequest["ReserveStockRequested<br/>event"]
    ReserveStock["Reserve Stock<br/>Inventory Service"]
    StockReserved["StockReserved<br/>event"]
    StockFailed["StockReservationFailed<br/>event"]
    ConfirmOrder["Confirm Order<br/>Order Service"]
    RejectOrder["Reject Order<br/>Order Service"]

    CreateOrder --> ReserveRequest
    ReserveRequest --> ReserveStock
    ReserveStock -->|"success"| StockReserved
    ReserveStock -->|"failure"| StockFailed
    StockReserved --> ConfirmOrder
    StockFailed --> RejectOrder
```

Current saga steps:

- Order service creates an order with `PENDING` status.
- Order service publishes `ReserveStockRequested` through its outbox.
- Inventory service consumes the event and tries to reserve stock.
- Inventory service publishes either `StockReserved` or `StockReservationFailed`.
- Order service consumes the result event.
- Order service changes the order to `CONFIRMED` or `FAILED`.

This is a choreography-based Saga because services react to events directly. There is no separate Saga orchestrator service yet.

## Main Flows

### Catalog Product Flow

```mermaid
sequenceDiagram
    autonumber
    participant Client
    participant Gateway as API Gateway
    participant Write as Catalog Write Service
    participant WriteDB as Postgres
    participant Rabbit as RabbitMQ
    participant Read as Catalog Read Service
    participant Mongo as MongoDB

    Client->>Gateway: Create product
    Gateway->>Write: gRPC CreateProduct
    Write->>WriteDB: Save product + outbox message
    Write-->>Gateway: Product created
    Gateway-->>Client: HTTP response

    Write->>WriteDB: Outbox worker reads pending event
    Write->>Rabbit: Publish ProductCreated
    Rabbit->>Read: Consume ProductCreated
    Read->>Mongo: Upsert product projection

    Client->>Gateway: Get products
    Gateway->>Read: gRPC GetProducts
    Read->>Mongo: Query products
    Read-->>Gateway: Products
    Gateway-->>Client: HTTP response
```

### Order Flow

```mermaid
sequenceDiagram
    autonumber
    participant Client
    participant Order as Order Service
    participant OrderDB as Order Postgres
    participant Rabbit as RabbitMQ
    participant Inventory as Inventory Service
    participant InventoryDB as Inventory Postgres

    Client->>Order: gRPC CreateOrder
    Order->>OrderDB: Save order + ReserveStockRequested outbox
    Order-->>Client: Order PENDING

    Order->>OrderDB: Outbox worker reads pending event
    Order->>Rabbit: Publish ReserveStockRequested
    Rabbit->>Inventory: Consume ReserveStockRequested

    Inventory->>InventoryDB: Lock stock rows with SELECT FOR UPDATE
    alt stock available
        Inventory->>InventoryDB: Reserve stock + StockReserved outbox
        Inventory->>Rabbit: Publish StockReserved
        Rabbit->>Order: Consume StockReserved
        Order->>OrderDB: Mark order CONFIRMED
    else stock not available
        Inventory->>InventoryDB: Save StockReservationFailed outbox
        Inventory->>Rabbit: Publish StockReservationFailed
        Rabbit->>Order: Consume StockReservationFailed
        Order->>OrderDB: Mark order FAILED
    end
```

## Shared Packages

Path:

```text
pkg/
```

Current shared packages:

- `configloader`: shared Viper configuration loading, environment binding, and `.env` loading.
- `logger`: shared Zap logger setup.
- `errs`: shared error helpers.

## Configuration

Each service has a `config/` folder with JSON configuration files for different environments.

The project uses:

- JSON config files for non-secret defaults.
- Environment variables for deployment-specific values and secrets.
- `.env` files for local development convenience.
- Viper to merge config files and environment variables.

Environment variables should override config file values.

## Infrastructure

Infrastructure is separated from application services.

Infrastructure Compose file:

```text
deployments/docker-compose.infrastructure.yml
```

Includes:

- Postgres
- RabbitMQ
- MongoDB

Services Compose file:

```text
deployments/docker-compose.services.yml
```

Includes:

- API Gateway
- Catalog Write Service
- Catalog Read Service
- Order Service
- Inventory Service

## Development Commands

Install local tools:

```bash
make install-tools
```

Start local infrastructure:

```bash
make dev-up
```

Stop local infrastructure:

```bash
make dev-down
```

Run migrations:

```bash
make migrate-up
```

Generate protobuf code:

```bash
make proto
```

Run a service with Air:

```bash
make run-order
make run-inventory
make run-catalog-write
make run-catalog-read
make run-api-gateway
```

Run checks:

```bash
make fmt
make test
make vet
```

Run E2E tests:

```bash
make test-e2e
```

Build Docker images:

```bash
make run-docker-build
```

Start services with Docker Compose:

```bash
make deploy-up
```

Stop services:

```bash
make deploy-down
```

## Testing

The project uses several testing levels:

- Unit tests for small handlers/services with fake or mocked dependencies.
- Repository integration tests with real database dependencies where useful.
- E2E tests for business flows across services.

Current E2E tests live in:

```text
services/test/e2e/
```

E2E tests are intended to run against the Docker Compose environment. They may write test data to Postgres and MongoDB, so use unique test data and clean volumes when needed.

## CI

The GitHub Actions workflow lives in:

```text
.github/workflows/ci.yml
```

CI currently runs:

- `gofmt` check
- `go vet`
- `go test`
- `go build`

E2E tests and deployment are intentionally separate from the normal CI path.

## Project Status

This is a learning and practice project. Some production-ready pieces are intentionally still evolving, such as full health checks, retry policy, deployment automation, observability, and complete E2E coverage.
