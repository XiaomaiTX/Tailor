FROM golang:1.27.3-alpine3.23 AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o /api ./cmd/api

FROM alpine:3.23.3
RUN apk add --no-cache ca-certificates libgcc
COPY --from=builder /api /api
EXPOSE 8080
ENTRYPOINT ["/api"]
