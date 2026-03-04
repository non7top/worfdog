# Worfdog - Simple Watchdog Service

A simple watchdog service written in Go that monitors services and can automatically restart them or reboot the system on failures.

## Features

- **Plugin-based architecture** for monitoring different service types
- **HTTPS/HTTP monitoring** - Check web endpoints for availability
- **Systemd service monitoring** - Monitor systemd unit status
- **MySQL connectivity monitoring** - Check database connectivity
- **Automatic service restart** - Restart failed services automatically
- **System reboot on failure** - Reboot the server when service restart fails
- **Reboot tracking** - Limit maximum number of reboots within a time window
- **INI configuration** - Simple and familiar configuration format
- **Dry run mode** - Test configuration without executing actions

## Installation

### From Release

Download the latest release from the [Releases page](https://github.com/non7top/worfdog/releases):

```bash
# Download static binary
wget https://github.com/non7top/worfdog/releases/download/v0.3.8/worfdog
sudo cp worfdog /usr/local/bin/
sudo chmod +x /usr/local/bin/worfdog
```

### From DEB Package (Recommended)

Single DEB package works on all Debian/Ubuntu systems:

```bash
# Download DEB package
wget https://github.com/non7top/worfdog/releases/download/v0.3.8/worfdog_0.3.8_all.deb

# Install
sudo dpkg -i worfdog_0.3.8_all.deb

# Enable and start service
sudo systemctl enable worfdog
sudo systemctl start worfdog
```

The DEB package includes:
- Static binary in `/usr/bin/worfdog`
- Systemd service unit
- Example configuration in `/usr/share/doc/worfdog/`

### Build from Source

```bash
go build -buildvcs=false -o worfdog .
sudo cp worfdog /usr/local/bin/
```

## Usage

```bash
# Run with default config paths
worfdog

# Run with specific config file
worfdog -config /path/to/worfdog.ini

# Set custom check interval
worfdog -interval 60s

# Dry run (log actions without executing)
worfdog -dry_run

# Show current status
worfdog -status

# Reset reboot counter
worfdog -reset-reboots

# Show version
worfdog -version
```

## Configuration

Create a `/etc/worfdog/worfdog.ini` file:

```ini
[worfdog]
# Initial delay before first health check (default: 30s)
initial_delay = 30

# Health check interval (default: 30s)
interval = 30

# Dry run mode: log actions without executing (default: false)
# dry_run = false

[reboot]
enabled = true
max_restarts = 3
max_reboots = 3
window_hours = 24
# sudo_password = your_sudo_password_here

[nginx]
type = systemd
unit = nginx

[apache]
type = systemd
unit = apache2
# Override max_restarts for this service only
max_restarts = 5

[webapp]
type = https
url = https://localhost:443/health
timeout = 10
# Skip TLS certificate verification (insecure)
# insecure_skip_verify = true
# Or specify acceptable certificate hostnames
tls_hostnames = s24.wr0.ru,localhost
# Retry failed checks before marking as CRITICAL
max_retries = 3

[api]
type = https
url = https://localhost:8443/api/status
timeout = 15
# Optional custom restart command
restart_cmd = systemctl restart api-service
```

### Configuration Options

#### [worfdog] Section

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `initial_delay` | int | 30 | Seconds to wait before first health check |
| `interval` | int | 30 | Seconds between health checks |
| `dry_run` | bool | false | Log actions without executing them |

#### [reboot] Section

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | false | Enable/disable automatic system reboot on persistent failures |
| `max_restarts` | int | 3 | Maximum service restart attempts before considering reboot |
| `max_reboots` | int | 3 | Maximum number of reboots allowed within the time window |
| `window_hours` | int | 24 | Time window (in hours) for counting reboots |
| `sudo_password` | string | - | Optional sudo password for reboot command |

#### Service Sections

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `type` | string | - | Service type: `systemd` or `https`/`http` (**required**) |
| `unit` | string | - | Systemd unit name (for systemd type) |
| `url` | string | - | URL to check (for https/http type) |
| `timeout` | int | 10 | Request timeout in seconds |
| `restart_cmd` | string | - | Custom restart command (overrides default) |
| `max_restarts` | int | 0 | Max restart attempts for this service (0 = use global default) |
| `insecure_skip_verify` | bool | false | Skip TLS certificate verification |
| `tls_hostnames` | string | - | Comma-separated list of acceptable TLS certificate hostnames |
| `max_retries` | int | 1 | Check retries before marking as CRITICAL |

## How It Works

1. **Health Checks**: Worfdog periodically checks all configured services
2. **Retry Logic**: Failed checks are retried `max_retries` times (at check intervals) before marking as CRITICAL
3. **Service Restart**: When a service fails, it attempts to restart it up to `max_restarts` times
4. **Verification**: After restart, it verifies the service is healthy
5. **System Reboot**: If restart fails or is not configured, it reboots the system (if enabled)
6. **Reboot Limits**: Tracks reboots to prevent reboot loops (max N reboots per M hours)

## Log Output

```
[worfdog] 2026/02/28 15:43:16 Registered plugin: nginx (type: systemd)
[worfdog] 2026/02/28 15:43:16 Waiting 5s before first check...
[worfdog] 2026/02/28 15:43:21 Version: dev
[worfdog] 2026/02/28 15:43:21 Reboot config: enabled=true, max_restarts=3, max_reboots=3, window_hours=24
[worfdog] 2026/02/28 15:43:21 Service [nginx]: type=systemd, timeout=10, max_restarts=0, max_retries=0
[worfdog] 2026/02/28 15:43:21 Starting watchdog with 1 plugins, check interval: 30s
[worfdog] 2026/02/28 15:43:21 Reboots in last 24 hours: 0/3
[worfdog] 2026/02/28 15:43:21 [nginx] CRITICAL: Service inactive: inactive (failure 1/1) - attempting recovery
```

## Systemd Service

The DEB package includes a systemd unit file. Enable and start:

```bash
sudo systemctl enable worfdog
sudo systemctl start worfdog
sudo systemctl status worfdog
```

View logs:
```bash
journalctl -u worfdog -f
```

## License

MIT
