FROM golang:1.26-alpine AS builder
COPY core/ /build/core/
COPY ratelimit-tokenbucket/ /build/ratelimit-tokenbucket/
WORKDIR /build/ratelimit-tokenbucket
RUN go mod download
RUN CGO_ENABLED=0 go build -o /ratelimit-tokenbucket ./cmd/module
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /ratelimit-tokenbucket /
ENTRYPOINT ["/ratelimit-tokenbucket"]
