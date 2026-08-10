import { EventsEmit } from './mockWailsRuntime.mjs'

const clone = (value) => JSON.parse(JSON.stringify(value))

const projects = [
  {
    id: 'atlas-api',
    name: 'atlas-api',
    path: '/Users/demo/workspaces/atlas-api',
    detectedType: 'go',
    command: 'go',
    args: ['run', 'main.go'],
    cwd: '',
    env: {},
    commands: [
      { id: 'go', name: 'go', command: 'go', args: ['run', 'main.go'], cwd: '', env: {}, isDefault: true },
      { id: 'test', name: 'test', command: 'go', args: ['test', './...'], cwd: '', env: {}, isDefault: false },
    ],
  },
  {
    id: 'atlas-worker',
    name: 'atlas-worker',
    path: '/Users/demo/workspaces/atlas-worker',
    detectedType: 'go',
    command: 'go',
    args: ['run', './cmd/worker'],
    cwd: '',
    env: {},
    commands: [{ id: 'go', name: 'go', command: 'go', args: ['run', './cmd/worker'], cwd: '', env: {}, isDefault: true }],
  },
  {
    id: 'northwind-billing',
    name: 'northwind-billing',
    path: '/Users/demo/workspaces/northwind-billing',
    detectedType: 'go',
    command: 'go',
    args: ['run', './cmd/server'],
    cwd: '',
    env: {},
    commands: [{ id: 'go', name: 'go', command: 'go', args: ['run', './cmd/server'], cwd: '', env: {}, isDefault: true }],
  },
  {
    id: 'northwind-reports',
    name: 'northwind-reports',
    path: '/Users/demo/workspaces/northwind-reports',
    detectedType: 'go',
    command: 'go',
    args: ['test', './...'],
    cwd: '',
    env: {},
    commands: [{ id: 'go', name: 'go', command: 'go', args: ['test', './...'], cwd: '', env: {}, isDefault: true }],
  },
  {
    id: 'control-plane',
    name: 'control-plane',
    path: '/Users/demo/workspaces/control-plane',
    detectedType: 'go',
    command: 'go',
    args: ['run', '.'],
    cwd: '',
    env: {},
    commands: [{ id: 'go', name: 'go', command: 'go', args: ['run', '.'], cwd: '', env: {}, isDefault: true }],
  },
  {
    id: 'signal-relay',
    name: 'signal-relay',
    path: '/Users/demo/workspaces/signal-relay',
    detectedType: 'go',
    command: 'go',
    args: ['run', './cmd/app'],
    cwd: '',
    env: {},
    commands: [{ id: 'go', name: 'go', command: 'go', args: ['run', './cmd/app'], cwd: '', env: {}, isDefault: true }],
  },
  {
    id: 'atstarter-site',
    name: 'atstarter',
    path: '/Users/demo/workspaces/atstarter',
    detectedType: 'node',
    command: 'npm',
    args: ['run', 'docs:dev'],
    cwd: 'site',
    env: {},
    commands: [
      { id: 'docs', name: 'docs', command: 'npm', args: ['run', 'docs:dev'], cwd: 'site', env: {}, isDefault: true },
      { id: 'frontend', name: 'frontend', command: 'npm', args: ['run', 'build'], cwd: 'frontend', env: {}, isDefault: false },
    ],
  },
  {
    id: 'compose-stack',
    name: 'compose-stack',
    path: '/Users/demo/workspaces/compose-stack',
    detectedType: 'compose',
    command: 'docker',
    args: ['compose', 'up'],
    cwd: '',
    env: {},
    commands: [{ id: 'compose', name: 'compose', command: 'docker', args: ['compose', 'up'], cwd: '', env: {}, isDefault: true }],
  },
]

let groups = [
  {
    id: 'daily-stack',
    name: 'Daily Stack',
    items: [
      { projectId: 'atlas-api', commandId: 'go' },
      { projectId: 'atlas-worker', commandId: 'go' },
      { projectId: 'northwind-billing', commandId: 'go' },
      { projectId: 'control-plane', commandId: 'go' },
      { projectId: 'signal-relay', commandId: 'go' },
    ],
  },
  {
    id: 'team-manage',
    name: 'Docs And Tools',
    items: [{ projectId: 'atstarter-site', commandId: 'docs' }],
  },
]

const statuses = new Map()
const logs = new Map()
const branches = new Map(projects.map((project) => [project.path, project.id === 'atstarter-site' ? 'main' : 'master']))
const FILES = new Map([
  ['cmd/main.go', `package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	fmt.Println("atlas-api listening on http://localhost:8080")
	_ = http.ListenAndServe(":8080", nil)
}
`],
  ['README.md', `# Atlas API

Synthetic project used by the AT Starter site demo.

- API: http://localhost:8080
- Health: /health
`],
  ['go.mod', `module example.test/atlas-api

go 1.24
`],
  ['internal/router.go', `package internal

func Routes() []string {
	return []string{"/health", "/v1/projects"}
}
`],
])

const containers = [
  {
    id: 'ctr-api',
    name: 'compose-stack-api-1',
    image: 'ghcr.io/example/atlas-api:dev',
    state: 'running',
    status: 'Up 12 minutes',
    compose: 'compose-stack',
    service: 'api',
    ports: ['8080:8080'],
    composeWorkingDir: '/Users/demo/workspaces/compose-stack',
  },
  {
    id: 'ctr-postgres',
    name: 'compose-stack-postgres-1',
    image: 'postgres:16',
    state: 'running',
    status: 'Up 12 minutes',
    compose: 'compose-stack',
    service: 'postgres',
    ports: ['5432:5432'],
    composeWorkingDir: '/Users/demo/workspaces/compose-stack',
  },
  {
    id: 'ctr-redis',
    name: 'redis-cache',
    image: 'redis:7',
    state: 'exited',
    status: 'Exited (0) 4 minutes ago',
    compose: '',
    service: '',
    ports: [],
  },
]

function commandsFor(project) {
  return project.commands && project.commands.length ? project.commands : [{
    id: 'default',
    name: 'Default',
    command: project.command,
    args: project.args || [],
    cwd: project.cwd || '',
    env: project.env || {},
    isDefault: true,
  }]
}

function defaultCommand(project) {
  return commandsFor(project).find((command) => command.isDefault) || commandsFor(project)[0]
}

function runIdFor(projectId, commandId) {
  const project = projects.find((item) => item.id === projectId)
  const command = commandsFor(project).find((item) => item.id === commandId) || defaultCommand(project)
  return `${projectId}:${command?.id || 'default'}`
}

function commandLine(command) {
  return [command.command, ...(command.args || [])].filter(Boolean).join(' ')
}

function ensureRun(runId) {
  if (!statuses.has(runId)) statuses.set(runId, { State: 'stopped', PID: 0, ExitCode: 0 })
  if (!logs.has(runId)) logs.set(runId, [])
}

for (const project of projects) {
  for (const command of commandsFor(project)) ensureRun(`${project.id}:${command.id || 'default'}`)
}

function seedRun(runId, status, lines) {
  statuses.set(runId, status)
  logs.set(runId, [...lines])
}

seedRun('atlas-api:go', { State: 'running', PID: 4201, ExitCode: 0 }, [
  '[runner] atlas-api -> go run main.go',
  '[runner] cwd /Users/demo/workspaces/atlas-api',
  '[go] compiling ./cmd/server',
  '[api] atlas-api listening on http://localhost:8080',
  '[api] GET /health 200 1ms',
])
seedRun('atlas-worker:go', { State: 'running', PID: 4202, ExitCode: 0 }, [
  '[runner] atlas-worker -> go run ./cmd/worker',
  '[worker] queue connected: local-jobs',
  '[worker] processed job sync-catalog in 42ms',
])
seedRun('atstarter-site:docs', { State: 'running', PID: 4210, ExitCode: 0 }, [
  '[runner] atstarter -> npm run docs:dev',
  '[vitepress] dev server running at http://localhost:5174/atstarter/',
])
seedRun('northwind-reports:go', { State: 'exited', PID: 0, ExitCode: 0 }, [
  '[runner] northwind-reports -> go test ./...',
  'ok example.test/northwind-reports 0.318s',
])

function setRunStatus(runId, status) {
  statuses.set(runId, status)
  EventsEmit('status:' + runId, {
    state: status.State,
    pid: status.PID || 0,
    exitCode: status.ExitCode || 0,
  })
}

function appendLog(runId, text) {
  const next = [...(logs.get(runId) || []), text]
  logs.set(runId, next)
  EventsEmit('log:' + runId, { text })
}

function idleUpdateState(extra = {}) {
  return {
    current: 'v0.5.13',
    latest: 'v0.5.13',
    available: false,
    notes: '',
    checking: false,
    downloading: false,
    downloadPct: 0,
    ready: false,
    error: '',
    assetUrl: 'https://github.com/example/atstarter/releases/tag/v0.5.13',
    assetSize: 0,
    canInstall: false,
    lastCheckAt: Date.now(),
    ...extra,
  }
}

export async function ListProjects() { return clone(projects) }
export async function ListGroups() { return clone(groups) }
export async function ListMissingProjectIDs() { return [] }
export async function GetWorkspaces() { return ['/Users/demo/workspaces', '/Users/demo/sandboxes'] }
export async function SetWorkspaces() {}
export async function PickDirectory() { return '/Users/demo/workspaces' }
export async function PickDirectoryFrom(defaultDir) { return defaultDir || '/Users/demo/workspaces' }

// 演示站没有真实文件系统,给所有目录返回同一份 scripts,让编辑对话框里的
// `npm run ` 补全能被看到。
const demoPackageScripts = [
  { name: 'build', script: 'vite build' },
  { name: 'dev', script: 'vite' },
  { name: 'docs:dev', script: 'vitepress dev docs' },
  { name: 'lint', script: 'eslint .' },
  { name: 'test', script: 'vitest run' },
]

export async function ListPackageScripts() { return clone(demoPackageScripts) }
export async function DockerAvailable() { return { available: true, version: 'Docker 28.0.0', reason: '' } }
export async function GetProjectBranch(path) { return branches.get(path) || 'main' }

export async function GetStatus(runId) {
  ensureRun(runId)
  return clone(statuses.get(runId))
}

export async function GetLogs(runId) {
  ensureRun(runId)
  return clone(logs.get(runId))
}

export async function SearchProjectFiles(_projectId, query, limit = 100) {
  const needle = String(query || '').trim().toLowerCase()
  if (!needle) return { matches: [], truncated: false }
  const paths = [
    'cmd/',
    'cmd/main.go',
    'README.md',
    'go.mod',
    'internal/',
    'internal/router.go',
  ]
  const matched = paths
    .filter((path) => path.toLowerCase().includes(needle) || path.split('/').filter(Boolean).pop()?.toLowerCase().includes(needle))
    .map((path) => ({
      path,
      name: path.split('/').filter(Boolean).pop() || path,
      isDir: path.endsWith('/'),
    }))
  return {
    matches: clone(matched.slice(0, limit)),
    truncated: matched.length > limit,
  }
}

export async function ClearLogs(runId) {
  logs.set(runId, [])
}

export async function StartProjectCommand(projectId, commandId) {
  const project = projects.find((item) => item.id === projectId)
  const command = commandsFor(project).find((item) => item.id === commandId) || defaultCommand(project)
  const runId = runIdFor(projectId, command?.id)
  setRunStatus(runId, { State: 'running', PID: 4200 + projects.findIndex((item) => item.id === projectId), ExitCode: 0 })
  appendLog(runId, `[runner] ${project.name} -> ${commandLine(command)}`)
  appendLog(runId, `[runner] cwd ${command.cwd || project.path}`)
}

export async function StopProjectCommand(projectId, commandId) {
  const runId = runIdFor(projectId, commandId)
  setRunStatus(runId, { State: 'stopped', PID: 0, ExitCode: 0 })
  appendLog(runId, '[runner] stopped')
}

export async function StartProject(projectId) {
  const project = projects.find((item) => item.id === projectId)
  return StartProjectCommand(projectId, defaultCommand(project).id)
}

export async function StopProject(projectId) {
  const project = projects.find((item) => item.id === projectId)
  return StopProjectCommand(projectId, defaultCommand(project).id)
}

export async function UpdateProjectCommands(projectId, name, commands) {
  const project = projects.find((item) => item.id === projectId)
  if (!project) return
  project.name = name || project.name
  project.commands = clone(commands || project.commands)
}

export async function UpdateProject(nextProject) {
  const index = projects.findIndex((item) => item.id === nextProject.id)
  if (index !== -1) projects[index] = clone(nextProject)
}

export async function AddProject(dir) {
  const name = String(dir || '').split('/').filter(Boolean).pop() || 'new-project'
  projects.push({
    id: name.toLowerCase().replace(/[^a-z0-9]+/g, '-'),
    name,
    path: dir,
    detectedType: 'node',
    command: 'npm',
    args: ['run', 'dev'],
    cwd: '',
    env: {},
    commands: [{ id: 'dev', name: 'dev', command: 'npm', args: ['run', 'dev'], cwd: '', env: {}, isDefault: true }],
  })
}

export async function RemoveProject(projectId) {
  const index = projects.findIndex((item) => item.id === projectId)
  if (index !== -1) projects.splice(index, 1)
}

export async function ResetProjects() {}

export async function SaveGroup(group) {
  const next = { ...clone(group), id: group.id || `group-${Date.now()}` }
  const index = groups.findIndex((item) => item.id === next.id)
  if (index === -1) groups.push(next)
  else groups[index] = next
}

export async function RemoveGroup(groupId) {
  groups = groups.filter((item) => item.id !== groupId)
}

export async function StartGroup(groupId) {
  const group = groups.find((item) => item.id === groupId)
  for (const item of group?.items || []) await StartProjectCommand(item.projectId, item.commandId)
}

export async function StopGroup(groupId) {
  const group = groups.find((item) => item.id === groupId)
  for (const item of group?.items || []) await StopProjectCommand(item.projectId, item.commandId)
}

export async function ScanWorkspaces() {
  return clone(projects.slice(0, 5))
}

export async function AddScanned(items) {
  for (const item of items || []) {
    if (!projects.some((project) => project.id === item.id)) projects.push(clone(item))
  }
}

export async function ListContainers() { return clone(containers) }
export async function StartContainer(id) {
  const item = containers.find((container) => container.id === id)
  if (item) item.state = 'running'
  EventsEmit('docker:state', clone(containers))
}
export async function StopContainer(id) {
  const item = containers.find((container) => container.id === id)
  if (item) item.state = 'exited'
  EventsEmit('docker:state', clone(containers))
}
export async function RestartContainer(id) {
  await StopContainer(id)
  await StartContainer(id)
}
export async function RemoveContainer(id) {
  const index = containers.findIndex((container) => container.id === id)
  if (index !== -1) containers.splice(index, 1)
  EventsEmit('docker:state', clone(containers))
}

export async function ListComposeServices() {
  return [
    { name: 'api', state: 'running', image: 'ghcr.io/example/atlas-api:dev', ports: ['8080:8080'] },
    { name: 'postgres', state: 'running', image: 'postgres:16', ports: ['5432:5432'] },
    { name: 'redis', state: 'exited', image: 'redis:7', ports: ['6379:6379'] },
  ]
}
export async function ComposeUp() {}
export async function ComposeStop() {}
export async function ComposeRestart() {}
export async function ComposeDown() {}
export async function FollowComposeLogs(projectId, service = '') {
  appendLog(service ? `compose:${projectId}:${service}` : `compose:${projectId}`, `[compose] following ${service || 'all services'}`)
}
export async function StopFollowComposeLogs() {}
export async function FollowContainerLogs(id) {
  appendLog(`container:${id}`, `[docker] following logs for ${id}`)
}
export async function StopFollowContainerLogs() {}

export async function UpdateGetState() { return idleUpdateState() }
export async function UpdateCheck() {
  const state = idleUpdateState()
  EventsEmit('update:state', state)
  return state
}
export async function UpdateStartDownload() { return idleUpdateState({ downloading: true, downloadPct: 35 }) }
export async function UpdateCancel() { return idleUpdateState() }
export async function UpdateInstall() { return idleUpdateState() }

export async function ListProjectDir(projectId, relPath = '') {
  if (relPath) {
    return [
      { name: 'main.go', isDir: false, size: 1240 },
      { name: 'README.md', isDir: false, size: 860 },
    ]
  }
  return [
    { name: 'cmd', isDir: true, size: 0 },
    { name: 'internal', isDir: true, size: 0 },
    { name: 'go.mod', isDir: false, size: 245 },
    { name: 'README.md', isDir: false, size: 860 },
  ]
}

function fileContent(path) {
  return FILES.get(path) || `# ${path || 'project'}

Synthetic file content for the AT Starter site demo.
`
}

export async function ReadProjectFile(projectId, path) {
  const content = fileContent(path || projectId)
  return { content, size: new TextEncoder().encode(content).length, truncated: false, binary: false }
}
export async function WriteProjectFile() {}
export async function ReadProjectFileBytes(projectId, path) {
  return { data: Array.from(new TextEncoder().encode(fileContent(path || projectId))), modTime: Date.now(), isBinary: false }
}
export async function WriteProjectFileBytes() { return Date.now() }
export async function ProjectAssetURL() { return '' }
export async function OpenProjectPath() {}
export async function ProjectFileMeta(projectId, path) {
  return { size: new TextEncoder().encode(fileContent(path || projectId)).length, modTime: Date.now(), isDir: false, isBinary: false }
}
export async function CreateProjectFile() {}
export async function MkdirProject() {}
export async function RenameProject() {}
export async function RemoveProjectPath() {}
export async function TrashProjectPath() {}
export async function WatchProjectDir() { return Date.now() }
export async function UnwatchProjectDir() {}
export async function UpdateProjectCommand() {}
