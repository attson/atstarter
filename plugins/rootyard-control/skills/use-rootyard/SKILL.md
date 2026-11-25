---
name: use-rootyard
description: Use when the user asks an AI agent to manage local rootyard projects, launch commands, groups, Docker containers, compose services, or logs.
---

# Use rootyard

Use the rootyard MCP tools first. They control the running desktop app through
the local control service. Do not edit `config.json` directly for runtime
operations.

## Startup

1. Call `rootyard_app_ping`.
2. If it returns `app_not_running`, call `rootyard_app_start`.
3. If MCP is unavailable but the `rootyard` binary is on PATH, use the CLI:

```bash
rootyard cli app start --wait
```

## Project workflow

1. Scan workspace roots with `rootyard_scan` and set `add=true` to save
   detected projects.
2. Add one known project directory with `rootyard_project_add`.
3. List projects with `rootyard_project_list`.
4. Inspect commands with `rootyard_project_commands`.
5. Inspect compose/cmd alternatives with `rootyard_project_detection_options`
   and switch with `rootyard_project_switch_type`.
6. Start or stop commands with `rootyard_project_start` /
   `rootyard_project_stop`.
7. Check state with `rootyard_project_status`.
8. Read logs with `rootyard_project_logs` and pass `tail` for concise output.

Targets can be project IDs or names. If a name is ambiguous, retry with the ID.
Omit `command` to use the default command.

## Group workflow

- Use `rootyard_group_create` to create a launch group with project command
  items.
- Use `rootyard_group_update` to rename a group or replace its items.
- Use `rootyard_group_add_item` / `rootyard_group_remove_item` for
  incremental membership changes.
- Use `rootyard_group_start` / `rootyard_group_stop` for runtime operations.

## Docker and compose

- Use `rootyard_docker_info` before Docker operations.
- Use `rootyard_container_list` for host containers.
- Use `rootyard_compose_services` before service-level compose operations.
- Use `rootyard_compose_logs` or `rootyard_container_logs` with `tail` when
  diagnosing failures.

## CLI fallback

All CLI output is JSON:

```bash
rootyard cli project list
rootyard cli scan ~/GolandProjects --add
rootyard cli project switch-type <project> --type go
rootyard cli group create dev --item api:default
rootyard cli group add-item dev --item worker:default
rootyard cli project start <project> --command default
rootyard cli project logs <project> --command default --tail 200
rootyard cli compose up <project> --service web
rootyard cli compose logs <project> --service web --tail 200
```

For a non-default config path, set `ROOTYARD_CONFIG` or pass `--config`.
