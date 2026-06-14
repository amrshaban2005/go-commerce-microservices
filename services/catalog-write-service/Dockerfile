FROM golang:1.26-alpine AS builder

WORKDIR /workspace

COPY pkg ./pkg
COPY api/gen/go ./api/gen/go
COPY services/catalog-write-service ./services/catalog-write-service

RUN go work init \
    ./pkg \
    ./api/gen/go \
    ./services/catalog-write-service

WORKDIR /workspace/services/catalog-write-service

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/catalog-write-service ./cmd


FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /out/catalog-write-service ./catalog-write-service
COPY --from=builder /workspace/services/catalog-write-service/config/*.json ./config/

ENV CONFIG_PATH=/app/config

EXPOSE 6005

ENTRYPOINT ["./catalog-write-service"]