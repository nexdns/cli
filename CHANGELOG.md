# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.2] - 2026-11-22

### Security

- Built with Go 1.26.11. Release 1.0.1 was built with Go 1.26.1, whose `net/http`
  and `crypto/x509` packages have since received security fixes; `go.mod` now names
  the toolchain, so release builds use the patched one.

### Changed

- `apply` prints a content change as `content <old> -> <new>`.
- `zone check` no longer makes an extra request for the zone's details.

### Fixed

- The README listed only `records.read` and `records.write` for the key that
  `acme hook` runs with. It also needs `zones.read` to find the zone.

## [1.0.1] - 2026-07-31

### Fixed

- `nexdns version` reports the real version when the CLI was installed with
  `go install`, using the module version and commit that Go records in the
  binary. It used to print `dev`.

## [1.0.0] - 2026-07-31

Initial public release.
