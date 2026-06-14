FROM golang:1.26-alpine AS builder

WORKDIR /workspace

COPY pkg ./pkg
COPY api/gen/go ./api/gen/go
COPY services/order-service ./services/order-service

RUN go work init \
    ./pkg \
    ./api/gen/go \
    ./services/order-service

WORKDIR /workspace/services/order-service

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/order-service ./cmd


FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /out/order-service ./order-service
COPY --from=builder /workspace/services/order-service/config/*.json ./config/

ENV CONFIG_PATH=/app/config

EXPOSE 6005

ENTRYPOINT ["./order-service"]