# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.1] - 2026-07-31

### Fixed

- `nexdns version` reports the real version when the CLI was installed with
  `go install`. That path applies no link flags, so the binary said `dev` and
  read as a broken build; it now falls back to the module version and commit
  that Go records in every binary.

## [1.0.0] - 2026-07-31

Initial public release.
