# Build Stage
FROM golang:latest AS builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o stress-service ./src

# Service Container
FROM alpine:latest
WORKDIR /app/

ENV GIN_MODE release
ENV PORT 3000

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/stress-service .

CMD ["/app/stress-service"]