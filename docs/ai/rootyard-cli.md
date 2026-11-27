# Rootyard CLI and MCP for AI agents

Rootyard exposes a local control surface for AI agents through the desktop app.
The desktop app remains the source of truth for projects, process state, logs,
Docker, and compose operations.

Project environments are the deliberate exception. Top-level `rootyard env`
uses the same file-backed environment service as the desktop and works while
the desktop app is stopped. It does not use the control server. File locks,
configuration revisions, and target fingerprints coordinate concurrent
desktop and CLI use.

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

`rootyard env --config /path/to/config.json ...` uses sibling
`environments.json` and `environment-state.json` files beside that config.
The former is authoritative; the latter is rebuildable state.

## Offline project environment CLI

Project environments manage the six repository-local Git keys below and five worktree-local Codex
keys:

```text
user.name
user.email
user.signingKey
commit.gpgSign
tag.gpgSign
gpg.format
model
model_reasoning_effort
sandbox_mode
approval_policy
web_search
```

The workflow is always preset/defaults -> directory rule -> scan -> diff ->
explicit apply. Saving configuration, scanning, resolving, diffing, and
running doctor never writes native Git or Codex configuration.

### Global and field flags

`rootyard env` accepts these global flags before or after the command:

```text
--config PATH   resolve environments.json beside this config.json
--json          emit the success result in an {ok,data} JSON envelope
```

Commands that edit or resolve fields accept repeatable session/field flags:

```text
--git-set key=value   explicitly set one supported key
--git-clear key       explicitly remove repository-local values
--git-inherit key     leave the value to an earlier layer or unmanaged scope
--codex-set key=value
--codex-clear key
--codex-inherit key
```

`inherit`, `set(value)`, and `clear` are distinct persisted intents. Omitting a
field does not mean clear. Signing booleans use `true` or `false`; `gpg.format`
uses `openpgp`, `ssh`, or `x509`.

### Defaults and presets

```bash
rootyard env defaults get
rootyard env defaults set --preset PRESET_ID
rootyard env defaults set --no-preset --git-set user.name="Ada Lovelace"

rootyard env preset list
rootyard env preset get PRESET_ID
rootyard env preset set --name "Company" --description "Company identity" \
  --git-set user.name="Ada Lovelace" \
  --git-set user.email=ada@company.example \
  --git-set commit.gpgSign=true
rootyard env preset set --id PRESET_ID --name "Company identity"
rootyard env preset set --name Company --codex-set model=custom-model
rootyard env preset clone PRESET_ID --name "Company signing"
rootyard env preset delete PRESET_ID
```

`preset set` creates when `--id` is absent and updates that stable ID when it
is present. A preset referenced by defaults or a rule cannot be deleted.

### Directory rules

```bash
rootyard env rule list
rootyard env rule get RULE_ID
rootyard env rule set --name "Company repositories" \
  --path /work/company --preset PRESET_ID --recursive --auto-discover
rootyard env rule set --id RULE_ID --exact --no-auto-discover --no-preset \
  --git-set user.email=team@example.com
rootyard env rule delete RULE_ID
```

Rule flags are:

```text
--id ID                    update an existing rule
--name NAME                set its display name
--path PATH                set its existing directory root
--preset ID | --no-preset  attach or detach one preset
--recursive | --exact      match descendants or only the rule path
--auto-discover | --no-auto-discover
```

Rule paths are canonicalized and unique. Auto-discovery runs at desktop
startup and after enabled rule saves; it does not grant permission to apply.

### Scan, resolve, diff, apply, and doctor

```bash
rootyard env scan
rootyard env scan --rule RULE_ID

rootyard env resolve /work/company/api
rootyard env resolve /work/company/api --git-set user.email=session@example.com

rootyard env diff /work/company/api
rootyard env diff --repo /work/company/api --repo /work/company/web
rootyard env diff --rule RULE_ID
rootyard env diff /work/company/api --codex-set sandbox_mode=workspace-write

rootyard env apply /work/company/api
rootyard env apply --rule RULE_ID --yes
rootyard env apply --repo /work/company/api --git-clear user.signingKey --yes
rootyard env apply /work/company/api --codex-clear model --yes
rootyard env apply /work/company/api --codex-set sandbox_mode=danger-full-access \
  --codex-set approval_policy=never --yes --accept-dangerous

rootyard env doctor /work/company/api
rootyard env doctor --rule RULE_ID
```

`scan --rule` is repeatable; no rule filter scans all rules. `diff` and
`apply` accept positional repository paths, repeatable `--repo PATH`, repeatable
`--rule ID`, and session field flags. `doctor` accepts paths and repeatable
`--rule ID`.

Interactive `apply` first prints the exact change plan and prompts
`Apply these changes? [y/N]`. It proceeds only for `y` or `yes`. `--json` and
any other non-interactive input require `--yes`; otherwise the command returns
`confirmation_required` and includes the unapplied plans. Apply uses the exact
fingerprint-bound plans it displayed and reports each target independently.
High-risk Codex plans also require typing `APPLY DANGEROUS` interactively or
passing `--accept-dangerous` non-interactively. Rootyard never edits global Codex
configuration and does not verify account or model availability.

### Worktrees and offline diagnostics

Linked worktrees that share a canonical Git common directory also share one
local config, so Rootyard de-duplicates them into one apply target while
retaining every path as an alias. If aliases match different rules and resolve
to different desired fields, `shared_target_conflict` blocks the target.
Each canonical worktree remains a separate Codex target with its own
`.codex/config.toml`.

When commit or tag signing resolves to true, `doctor` validates the configured
program or local fallback without network access:

| Format | Default program | Local check |
|--------|-----------------|-------------|
| `openpgp` | `gpg` | secret key lookup |
| `ssh` | `ssh-keygen` | readable key path and fingerprint lookup |
| `x509` | `gpgsm` | secret key lookup |

### Environment errors

| Code | Meaning / action |
|------|------------------|
| `environment_stale` | `environments.json` changed since it was loaded; reload before saving |
| `external_local_origin` | a managed value comes from an included local config; edit/detach that origin explicitly |
| `shared_target_conflict` | aliases of one common directory resolved differently; align their rules |
| `stale_plan` | native Git or Codex config changed after preview; run diff again before apply |
| `invalid_codex_config` | project Codex TOML is malformed or a managed value is invalid |
| `unsafe_codex_config_path` | `.codex` or its config is a symlink/non-regular path |
| `dangerous_codex_permissions` | sandboxing and approvals are both disabled; explicit high-risk confirmation is required |
| `codex_unavailable` | local Codex executable was not found; configuration may still be prepared |
| `lock_timeout` | another desktop/CLI mutation holds an environment file lock; retry after it completes |
| `signing_key_unavailable` | the resolved key is missing, unreadable, or failed local verification |
| `signing_program_unavailable` | the required signing program cannot be found |

Project-environment Git operations are local and timeout bounded. They never
run `fetch`, `pull`, `push`, `merge`, `commit`, or `stash`.

## Runtime CLI output

All `rootyard cli` output is JSON:

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
