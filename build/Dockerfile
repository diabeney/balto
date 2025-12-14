# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /balto ./cmd/balto

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=builder /balto /balto

EXPOSE 5500

ENTRYPOINT ["/balto"]
