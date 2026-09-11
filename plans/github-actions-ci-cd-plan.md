# GitHub Actions CI/CD Implementation Plan

## Agreed scope

This plan covers the initial CI/CD design for the current monorepo:

- One AWS staging environment.
- No production environment.
- Docker Compose on one EC2 instance for application and infrastructure containers.
- Amazon ECR for application images.
- Stored AWS IAM access keys in a GitHub `staging` environment.
- Semantic Git tags to trigger releases.
- No infrastructure as code.

## Stage 1 — Standardize service lifecycle and health checks

### Overview

Give every service a clear startup, readiness, and graceful-shutdown lifecycle. Preserve Fx in services that already use it: `fx.App.Run` handles operating-system signals, while lifecycle hooks start and stop each owned resource. Readiness is a simple lifecycle flag and is disabled before graceful shutdown begins.

### Changes and steps

1. Keep each `cmd/main.go` small: load local configuration, build the Fx application, and call `fx.App.Run`.
2. Use Fx lifecycle hooks consistently:
   - `OnStart` starts listeners and background workers.
   - `OnStop` gracefully stops and waits for the resource it owns.
   - Fx handles `SIGINT` and `SIGTERM` and applies its bounded shutdown context.
3. Use `fx.Shutdowner` only when a critical server or permanent consumer fails unexpectedly after startup, so that failure initiates the normal Fx shutdown sequence.
4. Replace the API gateway's blocking `gin.Run` call with an owned `http.Server` so it can shut down gracefully.
5. Define the health-check semantics:
   - Liveness means the process is running.
   - Readiness means application startup completed and shutdown has not begun.
6. Add lightweight HTTP management endpoints to every service container:
   - `GET /health/liveness` returns `200` while the process is alive and does not fail because a downstream dependency is unavailable.
   - `GET /health/readiness` returns `200` only after startup is complete and returns `503` while starting, shutting down, or unable to serve correctly.
7. Implement readiness as a fast in-memory lifecycle flag; do not call PostgreSQL, RabbitMQ, or other downstream dependencies from the endpoint.
8. Set readiness to false initially, true only after successful startup, and false immediately when shutdown begins.
9. Implement a consistent graceful-shutdown order:
   - Reject new work by setting readiness to false.
   - Gracefully stop HTTP and gRPC listeners.
   - Cancel and wait for background workers.
   - Close messaging connections, database pools, caches, and other clients.
   - Enforce an overall shutdown timeout and return shutdown errors.
10. Add focused tests for:
    - Liveness responses.
    - Ready and unready responses.
    - Readiness transitions during startup and shutdown.
    - Graceful server and worker termination.

## Stage 2 — Add runtime timeouts and dependency failure handling

### Overview

Bound every inbound request, outbound dependency call, message operation, health check, and shutdown path so a slow or unavailable dependency cannot make a service wait indefinitely. Propagate cancellation through contexts and define consistent failure behavior before CI and E2E tests rely on it.

### Implementation phases

1. **API gateway boundaries:** configure HTTP server limits, add an API request deadline, propagate it to gRPC clients, and return HTTP `504` for deadline expiry.
2. **Service and synchronous dependency boundaries:** add unary gRPC request deadlines, then bound PostgreSQL, MongoDB, Redis, Elasticsearch, and outbound gRPC operations within the request budget.
3. **Messaging boundaries:** add RabbitMQ publishing and confirmation deadlines, bound consumer processing, and define retry, acknowledgement, and dead-letter behavior.

### Changes and steps

1. Define timeout configuration by operation type instead of using one global timeout for everything.
2. Configure bounded HTTP server behavior for header reading, request processing, idle connections, and graceful shutdown, while accounting for any streaming endpoints.
3. Propagate the incoming request context through handlers, services, repositories, and downstream calls so cancellation stops unnecessary work.
4. Add explicit deadlines to outbound HTTP and gRPC calls and ensure an inner dependency deadline is shorter than the caller's overall request deadline.
5. Add bounded contexts to PostgreSQL queries, MongoDB operations, Redis commands, and Elasticsearch requests.
6. Add RabbitMQ publishing and publisher-confirmation timeouts.
7. Give message-consumer processing a bounded context and define the outcome for success, retryable failure, non-retryable failure, and timeout:
   - Acknowledge only successfully completed messages.
   - Negatively acknowledge or retry recoverable failures according to the queue policy.
   - Route exhausted or non-retryable messages to the appropriate dead-letter path.
8. Ensure retry loops use bounded attempts, backoff, and one overall deadline so retries cannot extend processing indefinitely.
9. Retry only operations that are safe or idempotent, because a timeout does not prove that the previous attempt made no change.
10. Use consistent protocol-level timeout results:
    - Appropriate HTTP timeout responses at the gateway.
    - gRPC `DeadlineExceeded` where the deadline expires.
    - Explicit message retry or dead-letter behavior for consumer timeouts.
11. Keep health endpoints fast and independent of downstream calls; readiness continues to reflect only application lifecycle state.
12. Add tests for context propagation, deadline expiry, cancellation, retry limits, and message acknowledgement behavior.
13. Start with conservative timeout values based on each operation's expected behavior; record them in configuration so the observability measurements can evaluate the chosen limits.

## Stage 3 — Stabilize E2E tests and container readiness

### Overview

Use the lifecycle and health contract from Stage 1 to make local containers and E2E tests deterministic. Remove false-success behavior, misleading port metadata, and fixed-delay startup assumptions.

### Changes and steps

1. Fix the catalog E2E test so it repeatedly checks the read model until:
   - The product appears, and the test succeeds.
   - The context deadline expires, and the test fails.
2. Review the order E2E flow for similar false-success or fixed-delay behavior.
3. Update `scripts/check-health.sh` to:
   - Wait with a bounded timeout.
   - Return a non-zero status if a service is unhealthy.
   - Print useful container status and logs on failure.
4. Add Docker Compose health checks that call each container's liveness or readiness endpoint as appropriate.
5. Replace fixed local startup and deployment sleeps with readiness polling.
6. Correct Docker port declarations:
   - API gateway: `8080`.
   - Catalog read: `6001`.
   - Catalog write: `6002`.
   - Order: `6005`.
   - Document the management health port used by each service.
   - Remove other unnecessary port declarations from services without business-facing inbound listeners.
7. Ensure Compose shutdown sends `SIGTERM` and gives services enough time to complete their bounded graceful shutdown before forced termination.
8. Run the local Compose environment, migrations, health checks, and both E2E flows to confirm the baseline is reliable.

## Stage 4 — Build the core pull-request CI

### Overview

Turn the existing workflow into a clear CI pipeline covering source quality, tests, vetting, linting, and compilation. This stage remains focused on Go and does not deploy anything.

### Changes and steps

1. Configure CI triggers for:
   - Pull requests targeting `master`.
   - Pushes to `master`.
   - Manual `workflow_dispatch` runs.
2. Add workflow concurrency so a newer commit cancels an obsolete run for the same pull request.
3. Keep default workflow permissions read-only.
4. Add explicit job timeouts.
5. Add or normalize Make targets for:
   - Formatting verification without modifying files.
   - `go vet`.
   - Linting.
   - Unit tests.
   - Compilation.
6. Add `golangci-lint` with a repository configuration file and a pinned tool version.
7. Organize the workflow into understandable jobs:
   - `quality`: formatting, vet, and lint.
   - `test`: unit tests for all applicable Go modules.
   - `build`: compile the five deployable applications.
8. Run build and test jobs only after the quality job succeeds.
9. Continue building all services when shared packages or API contracts change.
10. Upload useful test or coverage output when tests fail.
11. Configure the successful CI jobs as required checks for merging into `master`.

## Stage 5 — Add container and E2E validation to CI

### Overview

Extend CI from validating Go code to validating the actual containerized application. Pull requests must prove that all Docker images build and that the composed system can execute its business flows.

### Changes and steps

1. Define a Docker build matrix for:
   - API gateway.
   - Catalog read service.
   - Catalog write service.
   - Inventory service.
   - Order service.
2. Use Docker Buildx to build each image with `push: false`.
3. Add Docker build caching to reduce repeated build time.
4. Run Docker builds only after the Go quality, test, and build jobs succeed.
5. Add a separate E2E job that:
   - Builds or loads the required application images.
   - Starts the infrastructure containers.
   - Waits for infrastructure readiness.
   - Runs database migrations.
   - Starts application services.
   - Waits for application readiness.
   - Runs the catalog and order E2E tests.
6. Ensure the E2E job always shuts down its Compose environment.
7. Upload Docker Compose status and container logs when E2E execution fails.
8. Confirm that a broken Dockerfile, unhealthy service, or failed E2E flow prevents merging.

## Stage 6 — Prepare the manual AWS staging environment

### Overview

Create the single AWS staging environment manually. This environment will use one EC2 instance, Docker Compose for the complete stack, ECR for application images, and stored IAM access keys for GitHub Actions.

### Changes and steps

1. Select one AWS region for all staging resources.
2. Create five private ECR repositories, one for each deployable application.
3. Create an EC2 staging instance with enough memory for the application and infrastructure containers.
4. Attach persistent EBS storage for Docker volumes.
5. Configure the EC2 security group so:
   - The API gateway is the only publicly reachable application.
   - Database, Redis, MongoDB, RabbitMQ, Elasticsearch, and gRPC ports are not publicly exposed.
   - Administration uses AWS Systems Manager.
6. Attach an EC2 instance role allowing:
   - Systems Manager instance access.
   - Read access to the five ECR repositories.
7. Install and configure on EC2:
   - Docker Engine.
   - Docker Compose.
   - AWS CLI.
   - Git.
   - Migration tooling.
   - Systems Manager agent, if it is not already available.
8. Create the staging deployment directory and clone the repository there.
9. Store infrastructure and service environment files only on EC2. Do not commit or place database credentials in GitHub.
10. Create a dedicated IAM user for GitHub Actions with no console password.
11. Give the IAM user only the staging permissions needed to:
    - Authenticate and push to the five ECR repositories.
    - Send deployment commands to the staging EC2 instance.
    - Read the status of those deployment commands.
12. Create a GitHub Environment named `staging`.
13. Add GitHub Environment secrets:
    - `AWS_ACCESS_KEY_ID`.
    - `AWS_SECRET_ACCESS_KEY`.
14. Add non-secret GitHub Environment variables:
    - `AWS_REGION`.
    - `AWS_ACCOUNT_ID`.
    - EC2 instance ID.
    - ECR repository names.
15. Verify the credentials with a small manual GitHub Actions authentication test before building the release workflow.

## Stage 7 — Publish tagged releases to Amazon ECR

### Overview

Create the release portion of CD. A semantic Git tag will trigger verification, build the five images, and publish them to ECR. This stage produces deployable artifacts but does not update EC2.

### Changes and steps

1. Create a release workflow triggered by tags matching `v*.*.*`.
2. Validate that the tag follows the expected `vMAJOR.MINOR.PATCH` format.
3. Confirm that the tagged commit belongs to the `master` history.
4. Check out the exact tagged commit.
5. Run the core verification checks against that commit.
6. Load the stored AWS credentials from the GitHub `staging` environment.
7. Authenticate Docker with Amazon ECR.
8. Build the five Docker images.
9. Apply two tags to each image:
   - The Git version, such as `v0.1.0`.
   - The full Git commit SHA.
10. Push both tags to their corresponding ECR repositories.
11. Do not create or deploy a mutable `latest` tag.
12. Add a GitHub Actions summary showing:
    - Git tag.
    - Commit SHA.
    - Image names.
    - ECR image URIs.
    - Published Docker tags.
13. Test the release flow with the first staging tag and verify that all five images appear in ECR.

## Stage 8 — Deploy tagged releases to EC2 staging

### Overview

Complete CD by extending the tag workflow. After all images are published successfully, GitHub Actions will instruct the staging EC2 instance to deploy the same Git tag.

### Changes and steps

1. Add a `deploy-staging` job after the ECR publishing jobs.
2. Associate the deployment job with the GitHub `staging` environment.
3. Add staging deployment concurrency so only one deployment runs at a time.
4. Create a staging Compose override that:
   - Preserves the existing service configuration.
   - Replaces local application image names with ECR image URIs.
   - Uses the Git tag supplied by the deployment workflow.
   - Continues running PostgreSQL, MongoDB, Redis, RabbitMQ, and Elasticsearch as Docker containers.
5. Create a staging deployment script that accepts the release tag.
6. Through AWS Systems Manager, instruct EC2 to:
   - Fetch the repository tag.
   - Check out the exact tagged version.
   - Authenticate with ECR using the EC2 instance role.
   - Start or update infrastructure containers.
   - Wait for infrastructure readiness.
   - Run database migrations.
   - Pull the five tagged application images.
   - Start or update application containers.
   - Wait for application readiness.
7. Return a non-zero status to GitHub Actions if any deployment command or health check fails.
8. Run a post-deployment smoke test against the API gateway.
9. Print the deployed Git tag, commit SHA, container images, and health status in the workflow summary.
10. Perform a complete acceptance run:

    ```text
    Create tag
        -> CI verification
        -> Build images
        -> Push to ECR
        -> Deploy EC2 staging
        -> Run health checks
        -> Run smoke test
    ```

11. Document the operator commands for:
    - Creating and pushing a release tag.
    - Viewing the GitHub Actions run.
    - Inspecting ECR images.
    - Checking the staging containers.
    - Reading deployment and application logs.

## Stage 9 — Observability

### Overview

Measure service traffic, latency, failures, and asynchronous processing so slow or unhealthy behavior can be detected and investigated with metrics, traces, and correlated logs.

### Changes and steps

1. Add HTTP middleware and gRPC interceptors that record request count, error count, in-flight requests, and request-duration histograms.
2. Record latency by stable service, route template, gRPC method, and status labels without using request IDs or resource IDs as metric labels.
3. Calculate and display p50, p95, and p99 latency for HTTP and gRPC requests over explicit time windows.
4. Measure database query duration and database connection-pool usage.
5. Measure RabbitMQ publishing duration, consumer processing duration, failures, retries, and queue backlog.
6. Measure end-to-end business latency for order processing and catalog write-to-read projection.
7. Collect request rate, error rate, and latency together so latency is interpreted with its traffic volume and failure rate.
8. Expose Prometheus-compatible metrics and create Grafana dashboards for each service and the overall system.
9. Add structured request, correlation, and trace identifiers to logs and propagate them through HTTP, gRPC, and message metadata.
10. Add distributed traces so a slow request or message can be broken down by service, database, cache, and broker time.
11. correlation ID

## Stage 10 — Authentication and authorization through proxy forwarding

## Stage 11 - Rate limiting and Circut Breaker
