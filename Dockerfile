# Stage 1: download modules on native platform (avoids QEMU network issues)
FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS deps
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

# Stage 2: compile (may run under QEMU for arm64, but modules are already local)
FROM golang:1.22-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /src
COPY --from=deps /go/pkg/mod /go/pkg/mod
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/server ./cmd/server && \
    CGO_ENABLED=1 GOOS=linux go build -o /app/downsample ./cmd/downsample

FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/downsample .

RUN mkdir -p /app/data

EXPOSE 8080

CMD ["/app/server"]
