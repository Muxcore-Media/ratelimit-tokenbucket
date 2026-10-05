# Rate Limit Token Bucket

[![CI](https://github.com/Muxcore-Media/ratelimit-tokenbucket/actions/workflows/ci.yml/badge.svg)](https://github.com/Muxcore-Media/ratelimit-tokenbucket/actions)
[![Go Version](https://img.shields.io/badge/Go-1.26-blue)](https://go.dev/)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)

**Per-key token bucket rate limiter.**

A MuxCore sidecar module that enforces per-key token-bucket limits via gRPC (`Allow` / `Enabled`). Provides the `ratelimit` / `ratelimit.tokenbucket` capability.

---

## How It Works

```
Caller ──→ Allow(key) ──→ ratelimit-tokenbucket ──→ allow / deny
```

Each key gets its own bucket. Limiting is off until `RATELIMIT_ENABLED` is set.

---

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `RATELIMIT_GRPC_ADDR` | `127.0.0.1:9800` | gRPC listen address |
| `RATELIMIT_RATE` | `100` | Tokens replenished per second |
| `RATELIMIT_BURST` | `200` | Maximum bucket size |
| `RATELIMIT_ENABLED` | `false` | Enable limiting (`true` / `1`) |
| `RATELIMIT_TLS_CERT` / `KEY` / `CA` | auto | TLS material (or `MUXCORE_TLS_*`) |
| `MUXCORE_INSECURE_DISABLE_TLS` | — | Dev-only: disable inbound gRPC TLS |

---

## Quick Start

```bash
make build

export RATELIMIT_ENABLED=true
./ratelimit-tokenbucket --muxcore-mesh-addr localhost:9090
```

---

## Capability

`ratelimit` — Per-key token bucket rate limiter

## License

GPL-3.0
