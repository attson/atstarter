---
name: use-atstarter
description: Use when the user asks an AI agent to manage local atstarter projects, launch commands, groups, Docker containers, compose services, or logs.
---

# Use atstarter

Use the atstarter MCP tools first. They control the running desktop app through
the local control service. Do not edit `config.json` directly for runtime
operations.

## Startup

1. Call `atstarter_app_ping`.
2. If it returns `app_not_running`, call `atstarter_app_start`.
3. If MCP is unavailable but the `atstarter` binary is on PATH, use the CLI:

```bash
atstarter cli app start --wait
```

## Project workflow

1. Scan workspace roots with `atstarter_scan` and set `add=true` to save
   detected projects.
2. Add one known project directory with `atstarter_project_add`.
3. List projects with `atstarter_project_list`.
4. Inspect commands with `atstarter_project_commands`.
5. Inspect compose/cmd alternatives with `atstarter_project_detection_options`
   and switch with `atstarter_project_switch_type`.
6. Start or stop commands with `atstarter_project_start` /
   `atstarter_project_stop`.
7. Check state with `atstarter_project_status`.
8. Read logs with `atstarter_project_logs` and pass `tail` for concise output.

Targets can be project IDs or names. If a name is ambiguous, retry with the ID.
Omit `command` to use the default command.

## Group workflow

- Use `atstarter_group_create` to create a launch group with project command
  items.
- Use `atstarter_group_update` to rename a group or replace its items.
- Use `atstarter_group_add_item` / `atstarter_group_remove_item` for
  incremental membership changes.
- Use `atstarter_group_start` / `atstarter_group_stop` for runtime operations.

## Docker and compose

- Use `atstarter_docker_info` before Docker operations.
- Use `atstarter_container_list` for host containers.
- Use `atstarter_compose_services` before service-level compose operations.
- Use `atstarter_compose_logs` or `atstarter_container_logs` with `tail` when
  diagnosing failures.

## CLI fallback

All CLI output is JSON:

```bash
atstarter cli project list
atstarter cli scan ~/GolandProjects --add
atstarter cli project switch-type <project> --type go
atstarter cli group create dev --item api:default
atstarter cli group add-item dev --item worker:default
atstarter cli project start <project> --command default
atstarter cli project logs <project> --command default --tail 200
atstarter cli compose up <project> --service web
atstarter cli compose logs <project> --service web --tail 200
```

For a non-default config path, set `ATSTARTER_CONFIG` or pass `--config`.
