# Rate Limit Token Bucket

Per-key token bucket rate limiter for MuxCore.

Each key (e.g. IP, user ID, API key) has an independent token bucket. Tokens refill at `rate` per second up to `burst`. Rate limiting can be globally toggled.

## Configuration

| Env / Flag | Default | Description |
|---|---|---|
| `RATELIMIT_RATE` | `100` | Tokens added per second |
| `RATELIMIT_BURST` | `200` | Maximum burst size |
| `RATELIMIT_ENABLED` | `false` | Enable/disable rate limiting |
| `--grpc-addr` | `:9800` | gRPC listen address |

## RPCs

- `Allow(key)` — check if a request for the given key is allowed (consumes one token)
- `Enabled()` — returns whether rate limiting is currently enabled
