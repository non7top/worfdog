# Worfdog - Simple Watchdog Service

A simple watchdog service written in Go that monitors services and can automatically restart them or reboot the system on failures.

## Features

- **Plugin-based architecture** for monitoring different service types
- **HTTPS/HTTP monitoring** - Check web endpoints for availability
- **Systemd service monitoring** - Monitor systemd unit status
- **Automatic service restart** - Restart failed services automatically
- **System reboot on failure** - Reboot the server when service restart fails
- **Reboot tracking** - Limit maximum number of reboots within a time window
- **INI configuration** - Simple and familiar configuration format

## Installation

```bash
go build -buildvcs=false -o worfdog .
```

## Usage

```bash
# Run with default config paths
./worfdog

# Run with specific config file
./worfdog -config /path/to/worfdog.ini

# Set custom check interval
./worfdog -interval 60s

# Show current status
./worfdog -status

# Reset reboot counter
./worfdog -reset-reboots
```

## Configuration

Create a `worfdog.ini` file:

```ini
[reboot]
enabled = true
max_reboots = 3
window_hours = 24
# sudo_password = your_sudo_password_here

[nginx]
type = systemd
unit = nginx

[webapp]
type = https
url = https://localhost:443/health
timeout = 10

[api]
type = https
url = https://localhost:8443/api/status
timeout = 15
restart_cmd = systemctl restart api-service
```

### Configuration Options

#### Reboot Section
- `enabled` - Enable/disable automatic system reboot on persistent failures
- `max_reboots` - Maximum number of reboots allowed within the time window
- `window_hours` - Time window (in hours) for counting reboots
- `sudo_password` - Optional sudo password for reboot command

#### Service Sections
- `type` - Service type: `systemd` or `https`/`http`
- `unit` - Systemd unit name (for systemd type)
- `url` - URL to check (for https/http type)
- `timeout` - Request timeout in seconds
- `restart_cmd` - Optional custom restart command

## How It Works

1. **Health Checks**: Worfdog periodically checks all configured services
2. **Service Restart**: When a service fails, it attempts to restart it
3. **Verification**: After restart, it verifies the service is healthy
4. **System Reboot**: If restart fails and reboot is enabled, it reboots the system
5. **Reboot Limits**: Tracks reboots to prevent reboot loops

## License

MIT
