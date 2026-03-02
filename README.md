# Durable Task Scheduler (DTS) CLI

A k9s-style terminal UI for [Durable Task Scheduler](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler-overview) (DTS).

![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)

## Features

- **Orchestrations** — list, filter, sort, inspect, suspend, resume, terminate, restart, purge, raise events, create new
- **Entities** — list, inspect state, delete
- **Schedules** — list, create, pause, resume, delete
- **Workers** — view active/max counts with saturation bars
- **Agents** (preview) — start sessions, send prompts, view conversations
- **Multi-context** — kubectl-style named endpoints with quick switching
- **Azure AD auth** — DefaultAzureCredential, browser, CLI, device code
- **Vim navigation** — `j`/`k`, `:command`, `/filter`, `?` help
- **Title bar** — always-visible endpoint and task hub display
- **Auto-refresh** — configurable interval with countdown in status bar

## Install

```bash
go install github.com/microsoft/durabletask-scheduler/cli@latest
```

Or build from source:

```bash
go build -o dts-cli .
```

## Quick Start

```bash
# Launch with flags
dts-cli --url https://your-scheduler.durabletask.io --taskhub default

# Connect to the local DTS emulator (no auth, HTTP)
dts-cli --url http://localhost:8080 --taskhub default --auth-mode none

# Or configure a context first, then launch
dts-cli
# Use 'a' in Home view to add an endpoint
```

## Configuration

Config is stored at:
- **Windows**: `%APPDATA%\dts-cli\config.yaml`
- **Linux/macOS**: `~/.config/dts-cli/config.yaml`

```yaml
currentContext: my-dev
contexts:
  my-dev:
    url: https://my-dev-scheduler.durabletask.io
    taskHub: default
    tenantId: 00000000-0000-0000-0000-000000000000
settings:
  authMode: default    # default | browser | cli | device | none
  timeMode: local      # local | utc
  theme: dark          # dark | light
  refreshInterval: 30   # seconds (countdown shown in status bar)
  pageSize: 100
  enableAgents: true
  enableSchedules: true
```

## CLI Flags

| Flag | Description |
|------|-------------|
| `--url` | DTS endpoint URL (overrides current context) |
| `--taskhub` | Task hub name (overrides current context) |
| `--auth-mode` | Authentication: `default`, `browser`, `cli`, `device`, `none` |
| `--tenant-id` | Azure AD tenant ID |
| `--config` | Path to config file |
| `--no-splash` | Skip the splash screen |

## Keybindings

### Global

| Key | Action |
|-----|--------|
| `:` | Command mode |
| `/` | Filter |
| `?` | Help |
| `r` | Refresh (resets countdown) |
| `q` / `Esc` | Back / Quit |
| `Ctrl+C` | Quit |

### Commands

| Command | Description |
|---------|-------------|
| `:orch` | Go to Orchestrations |
| `:ent` | Go to Entities |
| `:sched` | Go to Schedules |
| `:work` | Go to Workers |
| `:ag` | Go to Agents |
| `:home` | Go to Home |
| `:help` | Show help |
| `:ctx <name>` | Switch context |
| `:q` | Quit (with confirmation) |
| `:q!` | Force quit (no confirmation) |

### Orchestrations List

| Key | Action |
|-----|--------|
| `Enter` | View detail |
| `o` | Cycle sort column |
| `O` | Toggle sort direction (asc/desc) |
| `1`-`5` | Quick status filter (All/Running/Completed/Failed/Pending) |
| `Space` | Toggle select |
| `a` | Select all |
| `s` | Suspend selected |
| `u` | Resume selected |
| `k` | Terminate selected |
| `Ctrl+K` | Force terminate |
| `x` | Restart selected |
| `p` | Purge selected |
| `n` | Create new |
| `[` / `]` | Previous / Next page |

### Orchestration Detail

| Key | Action |
|-----|--------|
| `Tab`/`BackTab` | Switch State/History tab |
| `s` | Suspend |
| `u` | Resume |
| `k` | Terminate |
| `Ctrl+K` | Force terminate |
| `x` | Restart |
| `w` | Rewind |
| `p` | Purge |
| `e` | Raise event |
| `i` | View input JSON |
| `o` | View output JSON |
| `c` | View custom status |

### Entities

| Key | Action |
|-----|--------|
| `Enter` | View detail |
| `d` | Delete selected |
| `Space` | Toggle select |
| `[` / `]` | Page |

### Schedules

| Key | Action |
|-----|--------|
| `Enter` | View JSON |
| `n` | Create new |
| `s` | Pause |
| `u` | Resume |
| `d` | Delete |
| `[` / `]` | Page |

### Agents

| Key | Action |
|-----|--------|
| `Enter` | Open session |
| `n` | Start new session |
| `d` | Delete session |
| `[` / `]` | Page |

## Authentication

dts-cli uses [Azure Identity](https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/azidentity) with scope `https://durabletask.io/.default`.

| Mode | Description |
|------|-------------|
| `default` | DefaultAzureCredential chain (env → managed identity → CLI → etc.) |
| `browser` | Interactive browser login |
| `cli` | Azure CLI credential (`az login`) |
| `device` | Device code flow |

## License

See repository root for license information.
