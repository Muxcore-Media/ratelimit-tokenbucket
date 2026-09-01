# Rate Limit Token Bucket

[![CI](https://git.zem.systems/muxcore/ratelimit-tokenbucket/actions/workflows/ci.yml/badge.svg)](https://git.zem.systems/muxcore/ratelimit-tokenbucket/actions)
[![Go Version](https://img.shields.io/badge/Go-1.26-blue)](https://go.dev/)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)

**Per-key token bucket rate limiter.**

A MuxCore sidecar module that enforces per-key token-bucket limits via gRPC (`Allow` / `Enabled`). Provides the `ratelimit` / `ratelimit.tokenbucket` capability and implements `contracts.RateLimiterProvider` for core HTTP middleware.

---

## How It Works

```
HTTP client ──→ muxcored (WireRateLimit) ──→ Allow(key) ──→ ratelimit-tokenbucket
```

Each key (typically client IP) gets its own **process-local** bucket inside this sidecar. Buckets are evicted after an idle TTL so unique keys do not grow memory forever.

**Fail-open by default:** limiting is off until `RATELIMIT_ENABLED=true`. When disabled, `Allow` always succeeds. The gRPC client used by core also **fails open** on RPC errors (allows the request).

**Core wiring required:** deploying this sidecar alone does not throttle API traffic. `muxcored` must discover the module and call `bootstrap.WireRateLimit` (automatic when the module registers with capability `ratelimit`). Without wiring, core keeps its built-in `DefaultRateLimiter`.

---

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `RATELIMIT_RATE` | `100` | Tokens replenished per second |
| `RATELIMIT_BURST` | `200` | Maximum bucket size |
| `RATELIMIT_ENABLED` | `false` | Enable limiting (`true` / `1`) |
| `RATELIMIT_IDLE_TTL` | `10m` | Drop idle per-key buckets after this duration |
| `RATELIMIT_GRPC_ADDR` | `127.0.0.1:9800` | gRPC listen address (loopback by default) |

Invalid `RATELIMIT_RATE`, `RATELIMIT_BURST`, or `RATELIMIT_IDLE_TTL` values are logged and ignored; defaults are kept.

---

## Quick Start

```bash
make build

export RATELIMIT_ENABLED=true
./ratelimit-tokenbucket --muxcore-mesh-addr localhost:9090
```

Ensure `muxcored` has registered this module so `WireRateLimit` replaces the default limiter.

---

## Capability

`ratelimit` — Per-key token bucket rate limiter (`RateLimiterProvider`)

## License

GPL-3.0
