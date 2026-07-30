# atstarter CLI and MCP for AI agents

atstarter exposes a local control surface for AI agents through the desktop app.
The desktop app remains the source of truth for projects, process state, logs,
Docker, and compose operations.

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
atstarter cli app start --wait
```

For dev or multiple configs, use either:

```bash
ATSTARTER_CONFIG=/path/to/config.json atstarter cli app ping
atstarter cli --config /path/to/config.json app ping
```

## CLI output

All CLI output is JSON:

```json
{"ok":true,"data":{}}
{"ok":false,"error":{"code":"app_not_running","message":"atstarter desktop app is not running","hint":"run: atstarter cli app start --wait"}}
```

`--follow` log commands emit JSON Lines, one envelope per update.

## Common commands

```bash
atstarter cli app ping
atstarter cli app start --wait

atstarter cli scan ~/GolandProjects ~/WebstormProjects
atstarter cli scan ~/GolandProjects --add
atstarter cli project add /path/to/project
atstarter cli project list
atstarter cli project commands <project>
atstarter cli project detection-options <project>
atstarter cli project switch-type <project> --type go
atstarter cli project switch-type <project> --type compose
atstarter cli project start <project> --command default
atstarter cli project stop <project> --command default
atstarter cli project restart <project> --command default --wait
atstarter cli project status <project> --command default
atstarter cli project logs <project> --command default --tail 200
atstarter cli project logs <project> --command default --tail 200 --follow
atstarter cli project logs-clear <project> --command default

atstarter cli group list
atstarter cli group create dev --item api:default --item web:serve
atstarter cli group update dev --name "local stack"
atstarter cli group add-item dev --item worker:default
atstarter cli group remove-item dev --item worker:default
atstarter cli group remove dev
atstarter cli group start <group>
atstarter cli group stop <group>

atstarter cli docker info
atstarter cli container list
atstarter cli container start <container>
atstarter cli container stop <container>
atstarter cli container restart <container>
atstarter cli container logs <container> --tail 200

atstarter cli compose services <project>
atstarter cli compose up <project> --service web
atstarter cli compose stop <project> --service web
atstarter cli compose restart <project> --service web
atstarter cli compose down <project>
atstarter cli compose logs <project> --service web --tail 200
```

Project, command, and group targets accept IDs or names. If a name matches more
than one item, the command fails and asks for an ID.

## MCP server

Run the stdio MCP server with:

```bash
atstarter mcp
```

The MCP server exposes tools with the `atstarter_` prefix, including:

- `atstarter_app_ping`
- `atstarter_app_start`
- `atstarter_scan`
- `atstarter_project_add`
- `atstarter_project_list`
- `atstarter_project_commands`
- `atstarter_project_detection_options`
- `atstarter_project_switch_type`
- `atstarter_project_start`
- `atstarter_project_stop`
- `atstarter_project_restart`
- `atstarter_project_status`
- `atstarter_project_logs`
- `atstarter_group_list`
- `atstarter_group_create`
- `atstarter_group_update`
- `atstarter_group_remove`
- `atstarter_group_add_item`
- `atstarter_group_remove_item`
- `atstarter_group_start`
- `atstarter_group_stop`
- `atstarter_docker_info`
- `atstarter_container_list`
- `atstarter_container_start`
- `atstarter_container_stop`
- `atstarter_container_restart`
- `atstarter_container_logs`
- `atstarter_compose_services`
- `atstarter_compose_up`
- `atstarter_compose_stop`
- `atstarter_compose_restart`
- `atstarter_compose_down`
- `atstarter_compose_logs`

MCP tool results are text content containing the same JSON envelope as the CLI.
