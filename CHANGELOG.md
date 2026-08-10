# Changelog

## [0.1.2] — 2026-08-10

### Added

- Advertise `settings` capability so admin-ui discovers SettingsProvider without ListAll probing.

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0]

### Added

- Per-key token-bucket rate limiter sidecar (`Allow` / `Enabled` gRPC)
- Capability `ratelimit`; defaults rate 100/s, burst 200, disabled until `RATELIMIT_ENABLED`
- Config via `RATELIMIT_RATE`, `RATELIMIT_BURST`, `RATELIMIT_ENABLED`; gRPC listen `:9800`
