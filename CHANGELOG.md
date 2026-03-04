# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- DEB package is now architecture-independent (`all` instead of `amd64`)
- Single DEB package works on all Debian/Ubuntu systems

## [0.3.9] - 2026-03-04

### Changed
- Build single `worfdog_<version>_all.deb` instead of separate Ubuntu 22.04/24.04 packages
- Binary is statically linked, works on any Linux distribution

### Added
- README.md included in `/usr/share/doc/worfdog/`
- Proper `postinst` script (creates config directory, copies example config)
- Proper `postrm` script (systemd daemon-reload on remove)
- `conffiles` for dpkg configuration tracking
- Improved package description

### Removed
- Separate `worfdog_*_noble_amd64.deb` package
- Separate `worfdog_*_jammy_amd64.deb` package

## [0.3.8] - 2026-03-04

### Added
- Combined `tag-and-release.yml` workflow for automatic releases on PR merge
- Automatic tag creation from PR body (`Tags vX.Y.Z`)
- Automatic release with assets and PR comments
- `AGENTS.md` - AI agent guide for the project

### Changed
- Release workflow triggered by PR merge (no manual tag push required)
- Fixed PR comment script to use `github.event.pull_request.number`

### Removed
- Separate `tag-on-pr-merge.yml` workflow (merged into release workflow)

## [0.3.7] - 2026-03-03

### Changed
- Dependabot security update: `filippo.io/edwards25519` 1.1.0 → 1.1.1

## [0.3.6] - 2026-03-03

### Changed
- Tag-on-pr-merge workflow now creates `v`-prefixed tags (e.g., `v0.3.6`)
- Release workflow accepts `v*` pattern for semantic versioning
- Updated PR comment templates to show correct tag format (`Tags vX.Y.Z`)

## [0.3.5] - 2026-03-03

### Fixed
- Action version: use `David-Lor/action-tag-on-pr-merge@0.0.3` (no `v` prefix)

## [0.3.2] - 2026-03-03

### Fixed
- Workflow Go version updated to 1.22 to match `go.mod` requirement

## [0.3.1] - 2026-03-03

### Added
- Reflection-based config key validation from struct tags
- Config validation warnings for unknown options in all sections
- Unit tests for config validation

### Changed
- `ValidKeys` map generated dynamically from struct `ini` tags
- Unknown config keys trigger warnings at startup

## [0.3.0] - 2026-03-03

### Added
- MySQL connectivity monitoring plugin
- Support for MySQL service type in configuration
- MySQL plugin uses `github.com/go-sql-driver/mysql` driver

### Changed
- GitHub Actions Go version updated to 1.22

## [0.2.1] - 2026-03-03

### Fixed
- Release workflow tag pattern: use simple `v*` glob (GitHub Actions limitation)
- SemVer validation handled by `tag-on-pr-merge` workflow regex

## [0.2.0] - 2026-02-28

### Added
- Configurable `initial_delay` via `[worfdog]` section
- Configurable `interval` via `[worfdog]` section
- Configurable `dry_run` via `[worfdog]` section
- Command line flags default to config file values

### Changed
- Default initial delay: 30s (matches default interval)
- All main options configurable via config file or CLI

## [0.1.2] - 2026-02-28

### Changed
- Services identified by presence of `type=` field in config sections
- Sections without `type=` are ignored (allows custom config sections)

## [0.1.1] - 2026-02-28

### Added
- Configurable initial delay via `[worfdog]` section (`initial_delay`)
- Configurable check interval via `[worfdog]` section (`interval`)
- Configurable dry run mode via `[worfdog]` section (`dry_run`)
- Command line flags now default to config file values

### Changed
- Default initial delay is now 30s (same as interval)
- All main options can be configured via config file or command line

## [0.1.0] - 2026-02-28

### Changed
- `max_retries` now uses check interval between retries (not hardcoded 2s)
- Retry log messages now use standard format with timestamps
- Recovery only triggered after `max_retries` consecutive failures

### Fixed
- Log format consistency across all messages

## [0.0.9] - 2026-02-28

### Added
- `max_retries` option for health check retries per service
- Retry logging for HTTPS and systemd plugins

## [0.0.8] - 2026-02-28

### Changed
- Log restart command when attempting service recovery
- Skip restart attempt if no restart command is configured
- Immediately consider reboot when restart command is missing

## [0.0.7] - 2026-02-28

### Added
- Version printed in log output at startup
- Config dump at startup (reboot config and service configs)
- Warning for unsupported/deprecated config options

### Changed
- Log format: `[worfdog]` without version in each message
- Version shown once at startup: `Version: 0.0.7`

## [0.0.6] - 2026-02-28

### Added
- `-dry_run` flag: log actions without executing
- Version in all log messages: `[worfdog v0.0.6]`
- Full test suite for build, flags, and config loading

## [0.0.5] - 2026-02-28

### Removed
- Per-check retry logic (replaced with interval-based retries)

## [0.0.4] - 2026-02-28

### Added
- `max_retries` for HTTPS health checks (retries before marking failed)

## [0.0.3] - 2026-02-28

### Added
- TLS configuration options:
  - `insecure_skip_verify`: Skip certificate verification
  - `tls_hostnames`: Accept certificates for specific hostnames

### Changed
- GitHub path updated to `non7top/worfdog`

## [0.0.2] - 2026-02-28

### Added
- Per-service `max_restarts` override
- Services can override global restart limit

## [0.0.1] - 2026-02-28

### Initial Release

### Features
- Systemd service monitoring
- HTTPS/HTTP endpoint monitoring
- Automatic service restart
- System reboot on persistent failures
- Reboot tracking and limits
- INI configuration file
- GitHub Actions CI/CD with:
  - Static binary builds (Linux amd64)
  - Single architecture-independent DEB package
  - Automated releases on PR merge

[Unreleased]: https://github.com/non7top/worfdog/compare/v0.3.9...HEAD
[0.3.9]: https://github.com/non7top/worfdog/compare/v0.3.8...v0.3.9
[0.3.8]: https://github.com/non7top/worfdog/compare/v0.3.7...v0.3.8
[0.3.7]: https://github.com/non7top/worfdog/compare/v0.3.6...v0.3.7
[0.3.6]: https://github.com/non7top/worfdog/compare/v0.3.5...v0.3.6
[0.3.5]: https://github.com/non7top/worfdog/compare/v0.3.2...v0.3.5
[0.3.2]: https://github.com/non7top/worfdog/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/non7top/worfdog/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/non7top/worfdog/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/non7top/worfdog/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/non7top/worfdog/compare/v0.1.2...v0.2.0
[0.1.2]: https://github.com/non7top/worfdog/releases/tag/v0.1.2
[0.1.1]: https://github.com/non7top/worfdog/releases/tag/v0.1.1
[0.1.0]: https://github.com/non7top/worfdog/releases/tag/v0.1.0
[0.0.9]: https://github.com/non7top/worfdog/releases/tag/v0.0.9
[0.0.8]: https://github.com/non7top/worfdog/releases/tag/v0.0.8
[0.0.7]: https://github.com/non7top/worfdog/releases/tag/v0.0.7
[0.0.6]: https://github.com/non7top/worfdog/releases/tag/v0.0.6
[0.0.5]: https://github.com/non7top/worfdog/releases/tag/v0.0.5
[0.0.4]: https://github.com/non7top/worfdog/releases/tag/v0.0.4
[0.0.3]: https://github.com/non7top/worfdog/releases/tag/v0.0.3
[0.0.2]: https://github.com/non7top/worfdog/releases/tag/v0.0.2
[0.0.1]: https://github.com/non7top/worfdog/releases/tag/v0.0.1
