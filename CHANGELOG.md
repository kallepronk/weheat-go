# Changelog

All notable changes to this project are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions follow [SemVer](https://semver.org).

## [Unreleased]

### Added
- Client with one method per WeHeat third-party endpoint.
- Keycloak token sources (password grant, refresh token) with rotation callback.
- Built-in sliding-window rate limiter, default 30 requests/hour.
