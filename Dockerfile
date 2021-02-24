# Build Stage
FROM golang:latest AS builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o service ./src

# Service Container
FROM alpine:latest
WORKDIR /app/

ENV GIN_MODE release
ENV PORT 8080

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/service .

CMD ["/app/service"]