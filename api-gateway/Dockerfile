FROM golang:1.26-alpine AS builder

WORKDIR /workspace

COPY pkg ./pkg
COPY api/gen/go ./api/gen/go
COPY api-gateway ./api-gateway

RUN go work init \
    ./pkg \
    ./api/gen/go \
    ./api-gateway

WORKDIR /workspace/api-gateway

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api-gateway ./cmd


FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /out/api-gateway ./api-gateway
COPY --from=builder /workspace/api-gateway/config/*.json ./config/

ENV APP_ENV=production
ENV CONFIG_PATH=/app/config

EXPOSE 6005

ENTRYPOINT ["./api-gateway"]