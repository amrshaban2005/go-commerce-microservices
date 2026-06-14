FROM golang:1.26-alpine AS builder

WORKDIR /workspace

COPY pkg ./pkg
COPY services/inventory-service ./services/inventory-service

RUN go work init \
    ./pkg \
    ./services/inventory-service

WORKDIR /workspace/services/inventory-service

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/inventory-service ./cmd


FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /out/inventory-service ./inventory-service
COPY --from=builder /workspace/services/inventory-service/config/*.json ./config/

ENV CONFIG_PATH=/app/config

EXPOSE 6005

ENTRYPOINT ["./inventory-service"]