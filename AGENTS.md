# AGENTS.md — ratelimit-tokenbucket

MuxCore sidecar module (`ratelimit-tokenbucket`). Workspace deploy and SSH: [`../AGENTS.md`](../AGENTS.md). Default ports: [`_mvp/PORTS.md`](../_mvp/PORTS.md).

## Module identity

| Field | Value |
|-------|-------|
| Directory | `ratelimit-tokenbucket` |
| Capabilities | `ratelimit`, `ratelimit.tokenbucket`, `settings` |
| Contracts | `RateLimiterProvider` (`core/pkg/contracts`); gRPC `RateLimitService` |

## Agent rules

- Modules run as gRPC sidecars; capabilities are the security boundary.
- TLS required in production (`MUXCORE_INSECURE_DISABLE_TLS` is dev-only).
- Match existing Go patterns; run `gofmt` and package tests before finishing.
- Cross-module events: prefer `github.com/Muxcore-Media/contracts-media/events` over deprecated `core/pkg/contracts` aliases.
- Do not edit polluted workspace dumps (see `MASTER-ROADMAP.md` Appendix H).
- Core must `WireRateLimit` or this sidecar does not throttle HTTP API traffic.

## Build

```bash
cd ratelimit-tokenbucket
nix-shell -p go --run 'go test ./...'
```
