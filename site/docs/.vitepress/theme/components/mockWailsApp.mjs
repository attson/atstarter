import { EventsEmit } from './mockWailsRuntime.mjs'

const clone = (value) => JSON.parse(JSON.stringify(value))

const projects = [
  {
    id: 'ad-advertising-platform',
    name: 'ad-advertising-platform',
    path: '/home/attson/GolandProjects/ad-advertising-platform',
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
    id: 'budget-usage-proxy',
    name: 'budget-usage-proxy',
    path: '/home/attson/GolandProjects/ad-ai-platform/budget-usage-proxy',
    detectedType: 'go',
    command: 'go',
    args: ['run', './cmd/proxy'],
    cwd: '',
    env: {},
    commands: [{ id: 'go', name: 'go', command: 'go', args: ['run', './cmd/proxy'], cwd: '', env: {}, isDefault: true }],
  },
  {
    id: 'material-tag-proxy',
    name: 'material-tag-proxy',
    path: '/home/attson/GolandProjects/ad-ai-platform/material-tag-proxy',
    detectedType: 'go',
    command: 'go',
    args: ['run', './cmd/server'],
    cwd: '',
    env: {},
    commands: [{ id: 'go', name: 'go', command: 'go', args: ['run', './cmd/server'], cwd: '', env: {}, isDefault: true }],
  },
  {
    id: 'feature-edits-multi-image',
    name: 'feature+edits_multi_image_20260624',
    path: '/home/attson/GolandProjects/ad-ai-toolkit/feature+edits_multi_image_20260624',
    detectedType: 'go',
    command: 'go',
    args: ['test', './...'],
    cwd: '',
    env: {},
    commands: [{ id: 'go', name: 'go', command: 'go', args: ['test', './...'], cwd: '', env: {}, isDefault: true }],
  },
  {
    id: 'master-cr',
    name: 'master_cr',
    path: '/home/attson/GolandProjects/ad-ai-toolkit/master_cr',
    detectedType: 'go',
    command: 'go',
    args: ['run', '.'],
    cwd: '',
    env: {},
    commands: [{ id: 'go', name: 'go', command: 'go', args: ['run', '.'], cwd: '', env: {}, isDefault: true }],
  },
  {
    id: 'rustling-drifting-trinket',
    name: 'rustling-drifting-trinket',
    path: '/home/attson/GolandProjects/ad-ai-toolkit/rustling-drifting-trinket',
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
    path: '/Users/attson/code/github.com.attson/atstarter',
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
    id: 'compose-local',
    name: 'docker-compose.yml',
    path: '/home/attson/GolandProjects/local-stack',
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
    id: 'material-tools',
    name: '素材打标',
    items: [
      { projectId: 'ad-advertising-platform', commandId: 'go' },
      { projectId: 'budget-usage-proxy', commandId: 'go' },
      { projectId: 'material-tag-proxy', commandId: 'go' },
      { projectId: 'master-cr', commandId: 'go' },
      { projectId: 'rustling-drifting-trinket', commandId: 'go' },
    ],
  },
  {
    id: 'team-manage',
    name: 'TeamManage',
    items: [{ projectId: 'atstarter-site', commandId: 'docs' }],
  },
]

const statuses = new Map()
const logs = new Map()
const branches = new Map(projects.map((project) => [project.path, project.id === 'atstarter-site' ? 'main' : 'master']))

const containers = [
  {
    id: 'ctr-api',
    name: 'local-stack-api-1',
    image: 'ghcr.io/attson/api:dev',
    state: 'running',
    status: 'Up 12 minutes',
    compose: 'local-stack',
    service: 'api',
    ports: ['8080:8080'],
    composeWorkingDir: '/home/attson/GolandProjects/local-stack',
  },
  {
    id: 'ctr-postgres',
    name: 'local-stack-postgres-1',
    image: 'postgres:16',
    state: 'running',
    status: 'Up 12 minutes',
    compose: 'local-stack',
    service: 'postgres',
    ports: ['5432:5432'],
    composeWorkingDir: '/home/attson/GolandProjects/local-stack',
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
    assetUrl: 'https://github.com/attson/atstarter/releases/tag/v0.5.13',
    assetSize: 0,
    canInstall: false,
    lastCheckAt: Date.now(),
    ...extra,
  }
}

export async function ListProjects() { return clone(projects) }
export async function ListGroups() { return clone(groups) }
export async function ListMissingProjectIDs() { return [] }
export async function GetWorkspaces() { return ['/home/attson/GolandProjects', '/Users/attson/code/github.com.attson'] }
export async function SetWorkspaces() {}
export async function PickDirectory() { return '/home/attson/GolandProjects' }
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
    { name: 'api', state: 'running', image: 'ghcr.io/attson/api:dev', ports: ['8080:8080'] },
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
export async function ReadProjectFile(projectId, path) {
  return { content: `# ${path || projectId}\n\nMock file content for the site demo.\n`, size: 64, truncated: false, binary: false }
}
export async function WriteProjectFile() {}
export async function ReadProjectFileBytes(projectId, path) {
  return { data: Array.from(new TextEncoder().encode(`# ${path || projectId}\n`)), modTime: Date.now(), isBinary: false }
}
export async function WriteProjectFileBytes() { return Date.now() }
export async function ProjectAssetURL() { return '' }
export async function OpenProjectPath() {}
export async function ProjectFileMeta() { return { size: 64, modTime: Date.now(), isDir: false, isBinary: false } }
export async function CreateProjectFile() {}
export async function MkdirProject() {}
export async function RenameProject() {}
export async function RemoveProjectPath() {}
export async function TrashProjectPath() {}
export async function WatchProjectDir() { return Date.now() }
export async function UnwatchProjectDir() {}
export async function UpdateProjectCommand() {}
