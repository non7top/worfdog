# Worfdog - AI Agent Guide

## Project Overview

**Worfdog** is a simple watchdog service written in Go that monitors services and can automatically restart them or reboot the system on failures.

**Repository:** https://github.com/non7top/worfdog

**Latest Release:** v0.4.0

## Quick Start

```bash
# Build
go build -buildvcs=false -o worfdog .

# Run with config
./worfdog -config /etc/worfdog/worfdog.ini

# Dry run (test without executing actions)
./worfdog -dry_run -config worfdog.ini
```

## Project Structure

```
worfdog/
├── main.go                      # Main application entry point
├── config/
│   └── config.go                # INI configuration loading & validation
├── plugins/
│   ├── plugin.go                # Plugin interface definition
│   ├── https.go                 # HTTPS/HTTP monitoring plugin
│   ├── systemd.go               # Systemd service monitoring plugin
│   ├── mysql.go                 # MySQL connectivity monitoring plugin
│   └── util.go                  # Utility functions
├── reboot/
│   └── tracker.go               # Reboot tracking with limits
├── .github/workflows/
│   ├── release.yml              # Build & release workflow
│   └── release-please.yml       # Release PR and tagging
├── worfdog.ini.example          # Example configuration
├── worfdog.service              # Systemd unit file
├── README.md                    # User documentation
├── CHANGELOG.md                 # Version history
└── worfdog_test.go              # Integration tests
```

## Configuration

### Config File Location
Loaded from (in order):
1. `-config` flag path
2. `worfdog.ini` (current directory)
3. `/etc/worfdog/worfdog.ini`
4. `/etc/worfdog.ini`

### Configuration Sections

#### [worfdog] - General Settings
```ini
[worfdog]
initial_delay = 30      # Seconds before first check (default: 30)
interval = 30           # Check interval in seconds (default: 30)
dry_run = false         # Log actions without executing (default: false)
```

#### [reboot] - Reboot Settings
```ini
[reboot]
enabled = true          # Enable automatic reboot on failure
max_restarts = 3        # Max restart attempts before reboot
max_reboots = 3         # Max reboots in window
window_hours = 24       # Time window for counting reboots
sudo_password = ""      # Optional sudo password for reboot
```

#### Service Sections
```ini
[service_name]
type = systemd          # or: https, http, mysql
unit = nginx            # For systemd type
url = https://...       # For https/http type
host = localhost        # For mysql type
port = 3306             # For mysql type
username = user         # For mysql type
password = secret       # For mysql type
database = db           # For mysql type
timeout = 10            # Timeout in seconds
restart_cmd = systemctl restart nginx
max_restarts = 5        # Override global max_restarts (0 = use global)
insecure_skip_verify = false  # Skip TLS verification
tls_hostnames = example.com   # Acceptable TLS cert hostnames
max_retries = 3         # Check retries before CRITICAL
```

### Valid Config Keys (Enforced)

The config loader validates keys using struct tags. Unknown keys trigger warnings.

**[worfdog]:** `initial_delay`, `interval`, `dry_run`

**[reboot]:** `enabled`, `max_restarts`, `max_reboots`, `window_hours`, `sudo_password`

**[service]:** `type`, `unit`, `url`, `host`, `port`, `username`, `password`, `database`, `timeout`, `restart_cmd`, `max_restarts`, `insecure_skip_verify`, `tls_hostnames`, `max_retries`

## Command Line Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-config` | Path to configuration file | Auto-detect |
| `-interval` | Health check interval | 30s (or config) |
| `-initial-delay` | Delay before first check | 30s (or config) |
| `-dry_run` | Log actions without executing | false (or config) |
| `-status` | Show status and exit | - |
| `-version` | Show version and exit | - |
| `-reset-reboots` | Reset reboot counter | - |

## Log Output Format

```
[worfdog] 2026/03/03 18:00:00 Version: 0.3.7
[worfdog] 2026/03/03 18:00:00 Reboot config: enabled=true, max_restarts=3, max_reboots=3, window_hours=24
[worfdog] 2026/03/03 18:00:00 Service [nginx]: type=systemd, timeout=10, max_restarts=0, max_retries=0
[worfdog] 2026/03/03 18:00:00 Starting watchdog with 3 plugins, check interval: 30s
[worfdog] 2026/03/03 18:00:00 Reboots in last 24 hours: 0/3
[worfdog] 2026/03/03 18:00:00 [nginx] OK: Service active
[worfdog] 2026/03/03 18:00:30 [webapp] CRITICAL: Connection failed: ... (failure 1/3)
[worfdog] 2026/03/03 18:01:00 [webapp] CRITICAL: Connection failed: ... (failure 2/3)
[worfdog] 2026/03/03 18:01:30 [webapp] CRITICAL: Connection failed: ... (failure 3/3) - attempting recovery
[worfdog] 2026/03/03 18:01:30 Attempting to restart service: webapp (attempt 1/3) using: systemctl restart webapp
[worfdog] 2026/03/03 18:01:30 Successfully restarted webapp
[worfdog] 2026/03/03 18:01:35 Service webapp recovered successfully
```

## Recovery Flow

1. **Health Check Fails** → Increment failure counter
2. **Failure Count >= max_retries** → Mark CRITICAL, attempt recovery
3. **Restart Attempt** → Use `restart_cmd` or default
4. **Restart Success** → Verify health, reset counters
5. **Restart Fail / No Command** → Consider reboot
6. **Reboot Check** → Verify limits, execute if allowed

## CI/CD Workflows

### release-please.yml / release-please-preview.yml

Thin callers of the shared workflows in `non7top/gh-workflows`. On every push to `master`, release-please maintains a release PR (version bump and `CHANGELOG.md` from conventional commits); merging it creates the `vX.Y.Z` tag and GitHub release. The preview workflow forecasts the bump on each PR. Config: `release-please-config.json`, `.release-please-manifest.json`.

Commit/PR title types decide the bump: `fix` = patch, `feat` = minor, `!`/`BREAKING CHANGE` = major. Dependabot uses `chore`, so bumps don't cut releases by themselves.

### release.yml

**Trigger:** `release-please` workflow completed, or manual `workflow_dispatch` with an existing tag.

Finds the release tag on the triggering commit, builds via `build-artifacts.yml` (static binary and one architecture-independent DEB) and attaches them to the release release-please already created. Tags created with `GITHUB_TOKEN` don't trigger `push: tags` workflows, which is why this runs after `release-please` instead.

**Release Assets:**
- `worfdog` - Static binary
- `worfdog_<version>_all.deb` - DEB package

## Testing

```bash
# Run all tests
go test -v ./...

# Test specific package
go test -v ./config/...

# Build and test binary
go build -buildvcs=false -o worfdog .
./worfdog -version
./worfdog -h
```

### Test Coverage

- `config/config_test.go` - Config validation tests
- `worfdog_test.go` - Binary build, flags, config loading tests

## Common Tasks for AI Agents

### Adding a New Plugin

1. Create `plugins/<name>.go`
2. Implement `Plugin` interface:
   ```go
   type Plugin interface {
       Name() string
       Check() CheckResult
       Restart() error
       GetConfig() config.ServiceConfig
   }
   ```
3. Add plugin type to `config.ServiceConfig`
4. Add valid keys to `config.ValidKeys`
5. Register in `main.go` plugin switch

### Adding Config Options

1. Add field to struct with `ini:"key_name"` tag
2. Add to `config.Load()` parsing
3. Update `ValidKeys` (auto-generated from struct tags via reflection)
4. Update documentation

### Creating a Release

1. Merge PRs with conventional-commit titles (`feat:`, `fix:`, ...)
2. Review and merge the release-please PR
3. Tag, release notes and assets are created automatically

### Branch and PR Workflow

**IMPORTANT:** Always follow this exact workflow for new PRs:

```bash
# 1. Start from updated master
git checkout master
git pull --rebase

# 2. Create new feature branch (NEVER reuse old branches)
git checkout -b feature/description-of-change

# 3. Make changes and commit
git add -A
git commit -m "Description of changes"

# 4. Push and create PR
git push -u origin feature/description-of-change
gh pr create --title "Title" --body "Description"

# 5. Leave PR open for review (NEVER merge autonomously)
```

**Rules:**
- ❌ Never merge PRs autonomously
- ❌ Never reuse old feature branches
- ❌ Never push directly to master
- ❌ Never create branches from other feature branches
- ✅ Always start from fresh `master` checkout
- ✅ Always create new branch for each PR
- ✅ Always leave merge decisions to repository owner

**Branch Cleanup:**
Feature branches are purged after PR merge. If a branch still exists:
```bash
# Delete local branch
git branch -d feature/old-branch

# Delete remote branch
git push origin --delete feature/old-branch
```

### Debugging Config Issues

```bash
# Check config validation warnings
./worfdog -config worfdog.ini 2>&1 | grep WARNING

# Test dry run
./worfdog -dry_run -config worfdog.ini

# Check loaded config
./worfdog -status -config worfdog.ini
```

## Version History

See [CHANGELOG.md](CHANGELOG.md) for full history.

**Recent Versions:**
- v0.3.9 - Single architecture-independent DEB package
- v0.3.8 - Combined tag-and-release workflow, AGENTS.md guide
- v0.3.7 - Dependabot security update
- v0.3.6 - v-prefixed tags for releases
- v0.3.5 - Action version fix (0.0.3)
- v0.3.2 - Go 1.22 workflow fix
- v0.3.0 - MySQL plugin added

## External Dependencies

| Package | Purpose |
|---------|---------|
| `gopkg.in/ini.v1` | INI config parsing |
| `github.com/go-sql-driver/mysql` | MySQL connectivity |

## Security Considerations

1. **Sudo Password** - Store in config file (restrict permissions: `chmod 600`)
2. **TLS Verification** - Use `tls_hostnames` instead of `insecure_skip_verify`
3. **Reboot Limits** - Prevents reboot loops (default: 3 per 24h)
4. **Dry Run Mode** - Test configuration safely before production

## Contact & Support

- **Issues:** https://github.com/non7top/worfdog/issues
- **Releases:** https://github.com/non7top/worfdog/releases
- **Actions:** https://github.com/non7top/worfdog/actions
