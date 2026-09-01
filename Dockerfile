FROM golang:1.26-alpine AS builder
COPY core/ /build/core/
COPY ratelimit-tokenbucket/ /build/ratelimit-tokenbucket/
WORKDIR /build/ratelimit-tokenbucket
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /ratelimit-tokenbucket ./cmd/module

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /ratelimit-tokenbucket /
EXPOSE 9800
HEALTHCHECK --interval=30s --timeout=5s --start-period=3s --retries=3 \
  CMD ["/ratelimit-tokenbucket", "--health-check"]
ENTRYPOINT ["/ratelimit-tokenbucket"]
