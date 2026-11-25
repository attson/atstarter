# Rootyard CLI and MCP for AI agents

Rootyard exposes a local control surface for AI agents through the desktop app.
The desktop app remains the source of truth for projects, process state, logs,
Docker, and compose operations.

## Installation and command name

The formal macOS, Linux, and Windows installers expose the same `rootyard`
command. On macOS, open the DMG and run `Rootyard.pkg`; on Linux, install the
Deb; on Windows, run the installer and open a new terminal. Portable tar/zip
archives do not modify PATH.

## Runtime model

When the desktop app starts, it opens a localhost-only control server and writes
a state file next to the config file:

```text
<config path>.control.json
```

The state file contains the local URL, a bearer token, the desktop app PID, and
the app version. File permissions are `0600`.

CLI and MCP clients read this state file and call the desktop app. If the desktop
app is not running, use:

```bash
rootyard cli app start --wait
```

For dev or multiple configs, use either:

```bash
ROOTYARD_CONFIG=/path/to/config.json rootyard cli app ping
rootyard cli --config /path/to/config.json app ping
```

## CLI output

All CLI output is JSON:

```json
{"ok":true,"data":{}}
{"ok":false,"error":{"code":"app_not_running","message":"rootyard desktop app is not running","hint":"run: rootyard cli app start --wait"}}
```

`--follow` log commands emit JSON Lines, one envelope per update.

## Common commands

```bash
rootyard cli app ping
rootyard cli app start --wait

rootyard cli scan ~/GolandProjects ~/WebstormProjects
rootyard cli scan ~/GolandProjects --add
rootyard cli project add /path/to/project
rootyard cli project list
rootyard cli project commands <project>
rootyard cli project detection-options <project>
rootyard cli project switch-type <project> --type go
rootyard cli project switch-type <project> --type compose
rootyard cli project start <project> --command default
rootyard cli project stop <project> --command default
rootyard cli project restart <project> --command default --wait
rootyard cli project status <project> --command default
rootyard cli project logs <project> --command default --tail 200
rootyard cli project logs <project> --command default --tail 200 --follow
rootyard cli project logs-clear <project> --command default

rootyard cli group list
rootyard cli group create dev --item api:default --item web:serve
rootyard cli group update dev --name "local stack"
rootyard cli group add-item dev --item worker:default
rootyard cli group remove-item dev --item worker:default
rootyard cli group remove dev
rootyard cli group start <group>
rootyard cli group stop <group>

rootyard cli docker info
rootyard cli container list
rootyard cli container start <container>
rootyard cli container stop <container>
rootyard cli container restart <container>
rootyard cli container logs <container> --tail 200

rootyard cli compose services <project>
rootyard cli compose up <project> --service web
rootyard cli compose stop <project> --service web
rootyard cli compose restart <project> --service web
rootyard cli compose down <project>
rootyard cli compose logs <project> --service web --tail 200
```

Project, command, and group targets accept IDs or names. If a name matches more
than one item, the command fails and asks for an ID.

## MCP server

Run the stdio MCP server with:

```bash
rootyard mcp
```

The MCP server exposes tools with the `rootyard_` prefix, including:

- `rootyard_app_ping`
- `rootyard_app_start`
- `rootyard_scan`
- `rootyard_project_add`
- `rootyard_project_list`
- `rootyard_project_commands`
- `rootyard_project_detection_options`
- `rootyard_project_switch_type`
- `rootyard_project_start`
- `rootyard_project_stop`
- `rootyard_project_restart`
- `rootyard_project_status`
- `rootyard_project_logs`
- `rootyard_group_list`
- `rootyard_group_create`
- `rootyard_group_update`
- `rootyard_group_remove`
- `rootyard_group_add_item`
- `rootyard_group_remove_item`
- `rootyard_group_start`
- `rootyard_group_stop`
- `rootyard_docker_info`
- `rootyard_container_list`
- `rootyard_container_start`
- `rootyard_container_stop`
- `rootyard_container_restart`
- `rootyard_container_logs`
- `rootyard_compose_services`
- `rootyard_compose_up`
- `rootyard_compose_stop`
- `rootyard_compose_restart`
- `rootyard_compose_down`
- `rootyard_compose_logs`

MCP tool results are text content containing the same JSON envelope as the CLI.

## AI plugins

rootyard ships a local plugin package under `plugins/rootyard-control`. The
same package is exposed through two marketplace manifests:

- `.agents/plugins/marketplace.json` for Codex.
- `.claude-plugin/marketplace.json` for Claude Code.

The plugin registers `rootyard mcp` and includes a `use-rootyard` skill that
teaches the agent to start the desktop app when needed, scan workspaces, manage
projects, groups, Docker, compose services, and logs.

### Codex

Install from the GitHub marketplace source:

```bash
codex plugin marketplace add attson/rootyard --ref main --sparse .agents --sparse plugins
codex plugin add rootyard-control@rootyard
```

Update later with:

```bash
codex plugin marketplace upgrade rootyard
codex plugin add rootyard-control@rootyard
```

Start a new Codex thread after installing or updating so the new skill and MCP
tools are loaded.

### Claude Code

Install from the GitHub marketplace source:

```bash
claude plugin marketplace add attson/rootyard --sparse .claude-plugin plugins
claude plugin install rootyard-control@rootyard
```

If the plugin is already installed, update later with:

```bash
claude plugin marketplace update rootyard
claude plugin update rootyard-control
```

Run `/reload-plugins` or start a new Claude Code session after installing or
updating.

### MCP fallback

If a client does not support plugins, register the MCP server directly:

```bash
claude mcp add rootyard -- rootyard mcp
```
