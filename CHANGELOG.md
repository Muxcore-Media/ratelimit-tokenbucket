# Changelog

## [0.1.2] — 2026-08-10

### Added

- Advertise `settings` capability so admin-ui discovers SettingsProvider without ListAll probing.

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.4] - 2026-10-05


### Changed
- Reported version comes from muxcore.json (ADR-0021); built on core v0.6.12 / sdk/go/module v0.6.3 (mesh enrollment, ADR-0017).

## [0.1.3] - 2026-10-05

### Changed
- CI runs on GitHub-hosted runners from the umbrella template; retired-origin workflows removed.
- Dependencies resolve from published GitHub tags (no filesystem `replace`); requires core v0.6.0.

## [0.1.3] — 2026-09-05

### Changed

- Inbound gRPC TLS enabled by default (`grpctls.ServerConfig` + `grpc.Creds`).
- Default listen address is loopback `127.0.0.1:9800` (was `:9800`); override with `RATELIMIT_GRPC_ADDR`.
- `MUXCORE_INSECURE_DISABLE_TLS` / `MUXCORE_GRPC_INSECURE` disable TLS for local development.

## [0.1.0]

### Added

- Per-key token-bucket rate limiter sidecar (`Allow` / `Enabled` gRPC)
- Capability `ratelimit`; defaults rate 100/s, burst 200, disabled until `RATELIMIT_ENABLED`
- Config via `RATELIMIT_RATE`, `RATELIMIT_BURST`, `RATELIMIT_ENABLED`; gRPC listen `127.0.0.1:9800`
