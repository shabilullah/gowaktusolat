# Stage 1: Build
FROM golang:1.26-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /server ./cmd/server

# Stage 2: Runtime
FROM alpine:3.21

COPY --from=builder /server /usr/local/bin/server

ENV DB_PATH=/data/waktusolat.db
ENV PORT=8080

EXPOSE 8080
CMD ["server"]
