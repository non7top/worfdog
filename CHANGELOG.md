# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.2] - 2026-02-28

### Changed
- Services are now identified by presence of `type=` field in config sections
- Any section without `type=` is ignored (allows custom config sections)

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
  - DEB packages for Ubuntu 22.04 and 24.04
  - Automated releases

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
