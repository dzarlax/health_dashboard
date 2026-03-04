# xx provides cross-compilation helpers (sets CC, GOARCH, etc.)
FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.6.1 AS xx

FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS builder
COPY --from=xx / /
ARG TARGETPLATFORM
RUN apk add --no-cache gcc musl-dev clang lld
RUN xx-apk add --no-cache gcc musl-dev
WORKDIR /src
COPY . .
RUN xx-go build -mod=vendor -o /app/server ./cmd/server && \
    xx-go build -mod=vendor -o /app/downsample ./cmd/downsample
RUN xx-verify /app/server && xx-verify /app/downsample

FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/downsample .

RUN mkdir -p /app/data

EXPOSE 8080

CMD ["/app/server"]
