# Durable Task Scheduler (DTS) CLI

A k9s-style terminal UI for [Durable Task Scheduler](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler-overview) (DTS).

![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)

## Features

- **Interactive TUI dashboard** — k9s-style terminal UI with multi-view navigation
- **Non-interactive CLI** — `dts-cli exec` command family for scripts and AI agents (JSON output)
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

### Interactive Dashboard (TUI)

The interactive dashboard includes a visual timeline view for orchestration details, showing a Gantt-style chart of activities, sub-orchestrations, timers, and events with colored duration bars.

```bash
# Launch with flags
dts-cli --url https://your-scheduler.durabletask.io --taskhub default

# Connect to the local DTS emulator (no auth, HTTP)
dts-cli --url http://localhost:8080 --taskhub default --auth-mode none

# Or configure a context first, then launch
dts-cli
# Use 'a' in Home view to add an endpoint
```

### Non-Interactive Commands (`exec`)

All `exec` commands return JSON to stdout. Errors are written as JSON to stderr with a non-zero exit code.

```bash
# Check connectivity
dts-cli exec ping --url https://my-scheduler.durabletask.io --taskhub default --auth-mode cli

# List orchestrations
dts-cli exec orchestrations list --url https://my-scheduler.durabletask.io --taskhub default

# Filter by status
dts-cli exec orch list --status Running,Failed --page-size 10

# Get orchestration detail
dts-cli exec orch get <instance-id>

# Get orchestration input/output/failure details
dts-cli exec orch payloads <instance-id>

# Get execution history
dts-cli exec orch history <instance-id>

# Create an orchestration
dts-cli exec orch create --name MyOrchestrator --input '{"key":"value"}'

# Suspend / Resume / Terminate
dts-cli exec orch suspend <instance-id> --reason "maintenance"
dts-cli exec orch resume <instance-id>
dts-cli exec orch terminate <instance-id> --reason "cancelled"

# Force-terminate multiple orchestrations
dts-cli exec orch force-terminate --ids id1,id2,id3 --reason "bulk cleanup"

# Restart / Rewind / Purge
dts-cli exec orch restart <instance-id>
dts-cli exec orch rewind <instance-id> --reason "retry after fix"
dts-cli exec orch purge <instance-id>

# Raise an event
dts-cli exec orch raise-event <instance-id> --event-name Approval --data '{"approved":true}'

# List entities
dts-cli exec entities list --name-starts-with MyEntity

# Get entity state
dts-cli exec ent state <instance-id>

# Delete entities
dts-cli exec ent delete <instance-id>

# List schedules
dts-cli exec schedules list

# Create a schedule
dts-cli exec sched create --schedule-id daily-job --orchestration-name MyOrch --interval PT24H

# Pause / Resume / Delete a schedule
dts-cli exec sched pause <schedule-id>
dts-cli exec sched resume <schedule-id>
dts-cli exec sched delete <schedule-id>

# List workers
dts-cli exec workers list

# List agent sessions
dts-cli exec agents list

# Start an agent session
dts-cli exec ag start --name MyAgent --session-id session1 --prompt "Hello"

# Send a prompt to an existing session
dts-cli exec ag send --name MyAgent --session-id session1 --prompt "What next?"

# Get agent session state
dts-cli exec ag state --name MyAgent --session-id session1

# Delete agent sessions
dts-cli exec ag delete @agent@MyAgent@session1
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

## Global Flags

These flags apply to both the interactive dashboard and all `exec` subcommands:

| Flag | Description |
|------|-------------|
| `--url` | DTS endpoint URL (overrides current context) |
| `--taskhub` | Task hub name (overrides current context) |
| `--auth-mode` | Authentication: `default`, `browser`, `cli`, `device`, `none` |
| `--tenant-id` | Azure AD tenant ID |
| `--config` | Path to config file |

### TUI-only Flags

| Flag | Description |
|------|-------------|
| `--no-splash` | Skip the splash screen |

## `exec` Command Reference

### `exec ping`

Check connectivity to the DTS backend.

### `exec orchestrations` (alias: `orch`)

| Subcommand | Arguments / Flags | Description |
|------------|-------------------|-------------|
| `list` | `--status`, `--name`, `--instance-id`, `--created-after`, `--created-before`, `--page-size`, `--start-index`, `--sort-by`, `--sort-dir` | List orchestrations with filters |
| `get <id>` | | Get orchestration metadata |
| `payloads <id>` | | Get input/output/failure details |
| `history <id>` | `--execution-id` | Get execution history (auto-detects execution ID) |
| `create` | `--name` (required), `--instance-id`, `--input`, `--version`, `--scheduled-start`, `--tags` | Create a new orchestration |
| `suspend <id>` | `--reason` | Suspend a running orchestration |
| `resume <id>` | `--reason` | Resume a suspended orchestration |
| `terminate <id>` | `--reason` | Terminate an orchestration |
| `force-terminate` | `--ids` (required), `--reason` | Force-terminate multiple orchestrations |
| `restart <id>` | `--new-id` | Restart an orchestration |
| `rewind <id>` | `--reason` | Rewind a failed orchestration |
| `purge <id> [id...]` | | Purge (delete) one or more orchestrations |
| `raise-event <id>` | `--event-name` (required), `--data` | Send a named event |

### `exec entities` (alias: `ent`)

| Subcommand | Arguments / Flags | Description |
|------------|-------------------|-------------|
| `list` | `--name`, `--name-starts-with`, `--page-size`, `--start-index` | List entities with filters |
| `get <id>` | | Get entity metadata |
| `state <id>` | | Get serialized entity state |
| `delete <id> [id...]` | | Delete one or more entities |

### `exec schedules` (alias: `sched`)

| Subcommand | Arguments / Flags | Description |
|------------|-------------------|-------------|
| `list` | `--continuation-token` | List schedules |
| `create` | `--schedule-id` (required), `--orchestration-name` (required), `--interval` (required), `--input`, `--instance-id`, `--start-at`, `--end-at`, `--start-immediately-if-late` | Create a schedule |
| `delete <id>` | | Delete a schedule |
| `pause <id>` | | Pause a schedule |
| `resume <id>` | | Resume a paused schedule |

### `exec workers` (alias: `work`)

| Subcommand | Arguments / Flags | Description |
|------------|-------------------|-------------|
| `list` | | List connected workers |

### `exec agents` (alias: `ag`)

| Subcommand | Arguments / Flags | Description |
|------------|-------------------|-------------|
| `list` | `--page-size`, `--start-index` | List agent sessions |
| `start` | `--name` (required), `--session-id` (required), `--prompt` (required) | Start a new agent session |
| `send` | `--name` (required), `--session-id` (required), `--prompt` (required) | Send a prompt to an existing session |
| `state` | `--name` (required), `--session-id` (required) | Get agent session state |
| `delete <id> [id...]` | | Delete agent session entities |

## Keybindings (Interactive Dashboard)

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
| `none` | No authentication (for local emulator) |

## License

See repository root for license information.

# Project

> This repo has been populated by an initial template to help get you started. Please
> make sure to update the content to build a great experience for community-building.

As the maintainer of this project, please make a few updates:

- Improving this README.MD file to provide a great experience
- Updating SUPPORT.MD with content about this project's support experience
- Understanding the security reporting process in SECURITY.MD
- Remove this section from the README

## Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit [Contributor License Agreements](https://cla.opensource.microsoft.com).

When you submit a pull request, a CLA bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., status check, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

## Trademarks

This project may contain trademarks or logos for projects, products, or services. Authorized use of Microsoft
trademarks or logos is subject to and must follow
[Microsoft's Trademark & Brand Guidelines](https://www.microsoft.com/legal/intellectualproperty/trademarks/usage/general).
Use of Microsoft trademarks or logos in modified versions of this project must not cause confusion or imply Microsoft sponsorship.
Any use of third-party trademarks or logos are subject to those third-party's policies.
