# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Rate limit proto definition and generated Go code
- Per-key token bucket rate limiter with configurable rate/burst
- Global enable/disable toggle (`Allow` returns true when disabled)
- Token refill: each key refills at `rate` tokens/sec up to `burst`
- Empty key validation returns InvalidArgument
- Full test suite: allow/exhaust/disable/empty-key/per-key
- Contract declaration with `MinCoreVersion: 0.4.0`

### Changed

- Makefile/Dockerfile/docker-compose/systemd: your-module → ratelimit-tokenbucket
