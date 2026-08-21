# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Releases before 1.0.0 are documented only in the
[GitHub releases](https://github.com/italia/developers-italia-api/releases).

## [Unreleased]

### Added

- A security policy, `SECURITY.md`, saying how to report a vulnerability
  privately and which versions get fixes.

## [1.4.0] - 2026-08-19

### Added

- An `analysis` field on catalogs, so a catalog carries the same
  analysis data software already did.

### Fixed

- The migration of the `logs` table failed on a column type mismatch,
  which left an upgraded database unusable.

### Security

- The release pipeline now pins every GitHub action by digest, denies
  workflow permissions by default, and verifies in CI that the released
  binary can be rebuilt byte for byte from source. Each release ships an
  SBOM and a build provenance attestation.
- The development image is pinned by digest.

## [1.3.0] - 2026-05-08

### Added

- Scopes on catalogs, so a token can be limited to one catalog.
- A root catalog, reachable through the `∅` alternativeId, which lets
  the catalog routes address the whole collection.
- Debouncing of webhook notifications, so a burst of writes produces one
  delivery instead of one per write. Tunable with `WEBHOOK_DEBOUNCE_MS`
  and `WEBHOOK_DEBOUNCE_MAX_MS`.
- Size and range limits on catalog sources, analysis values and JSON
  Patch bodies, rejected at validation instead of reaching the database.
- Stricter validation of publisher CodeHosting URLs.

### Changed

- Indexes added on the foreign key columns, which speeds up the list
  endpoints on a large catalog.

### Fixed

- Webhook dispatch had no timeout and could hold a request open
  indefinitely. It is now capped at 10 seconds.
- Every webhook was signed with the same secret rather than its own, so
  a receiver could verify a payload meant for a different endpoint.
- Pagination links dropped the query parameters of the original request,
  so following `next` silently widened the result set.
- A `fiber.Error` was classified as an internal error, turning a 4xx
  into a 500.
- Several OpenAPI defects: invalid examples, duplicate description keys
  and the wrong media type on the analysis PATCH endpoint.

## [1.2.0] - 2026-04-22

### Added

- A catalogs resource, with a `publishersNamespace` that scopes
  publisher ids to one catalog.
- `POST /catalogs/:id/logs`.
- A `token` subcommand that creates API tokens, so a deployment no
  longer needs a separate tool to issue one.

### Changed

- The project is now called software-catalog-api. The binary, the
  OpenAPI file and the documentation use the new name. The old binary
  name stays as a symlink, so existing deployments keep working.

### Removed

- The `analysis` payload no longer ships inside the software object. It
  moved to `GET /software/:id/analysis`, which keeps the software
  response small. Clients reading `software.analysis` must follow the
  new endpoint.

### Fixed

- The `url` query filter was ignored when listing the software of a
  catalog.
- Filtering by url returned inactive software.

## [1.1.0] - 2026-03-22

### Added

- A healthcheck endpoint, `/livez`, for readiness probes.
- JSON Patch support on the Publishers resource.
- Normalization of the URLs stored on publishers and software, so the
  same repository submitted in two spellings resolves to one record.

### Fixed

- A duplicate key when patching a publisher returned 500 instead of 409,
  and the 409 response now names the field that collided.
- The check for a conflicting `alternativeId` was not atomic, so two
  concurrent writes could both pass it.
- The paginator configuration was mutated globally, so one request could
  change the page size seen by another.

## [1.0.1] - 2026-02-25

### Changed

- Dependency and CI updates only, no behaviour change. Includes
  `golang.org/x/crypto` 0.36.0 to 0.45.0 and Fiber 2.52.9 to 2.52.12.

## [1.0.0] - 2025-09-09

First stable release. The API had already been running in production for
some time, and this tag declares the interface stable.

### Changed

- Go 1.24.

[Unreleased]: https://github.com/italia/developers-italia-api/compare/v1.4.0...HEAD
[1.4.0]: https://github.com/italia/developers-italia-api/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/italia/developers-italia-api/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/italia/developers-italia-api/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/italia/developers-italia-api/compare/v1.0.1...v1.1.0
[1.0.1]: https://github.com/italia/developers-italia-api/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/italia/developers-italia-api/compare/v0.12.1...v1.0.0
