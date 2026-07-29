const BASE_PROJECTS = [
  {
    id: 'web',
    name: 'web-admin',
    kind: 'Vue',
    path: '~/code/acme/web-admin',
    branch: 'feature/dashboard',
    status: 'running',
    port: '5173',
    commands: [
      { id: 'dev', label: 'dev', command: 'pnpm dev --host 0.0.0.0' },
      { id: 'build', label: 'build', command: 'pnpm build' },
      { id: 'test', label: 'test', command: 'pnpm vitest run' },
    ],
    files: ['src/App.vue', 'src/views/Dashboard.vue', 'src/stores/projects.ts', 'package.json'],
  },
  {
    id: 'api',
    name: 'api-server',
    kind: 'Go',
    path: '~/code/acme/api-server',
    branch: 'main',
    status: 'running',
    port: '8080',
    commands: [
      { id: 'dev', label: 'dev', command: 'air -c .air.toml' },
      { id: 'test', label: 'test', command: 'go test ./...' },
      { id: 'build', label: 'build', command: 'go build ./cmd/api' },
    ],
    files: ['cmd/api/main.go', 'internal/http/router.go', 'internal/store/project.go', 'go.mod'],
  },
  {
    id: 'docs',
    name: 'docs-site',
    kind: 'VitePress',
    path: '~/code/acme/docs-site',
    branch: 'main',
    status: 'stopped',
    port: '5174',
    commands: [
      { id: 'dev', label: 'dev', command: 'npm run docs:dev' },
      { id: 'build', label: 'build', command: 'npm run docs:build' },
    ],
    files: ['docs/index.md', 'docs/guide/index.md', '.vitepress/config.mjs', 'package.json'],
  },
  {
    id: 'compose',
    name: 'docker-compose.yml',
    kind: 'Compose',
    path: '~/code/acme/infra',
    branch: 'main',
    status: 'stopped',
    port: '5432',
    commands: [
      { id: 'up', label: 'up', command: 'docker compose up' },
      { id: 'down', label: 'down', command: 'docker compose down' },
      { id: 'logs', label: 'logs', command: 'docker compose logs -f' },
    ],
    files: ['docker-compose.yml', '.env.local', 'postgres/init.sql', 'redis/redis.conf'],
  },
]

const LOGS_BY_PROJECT = {
  web: [
    '[web-admin] pnpm dev --host 0.0.0.0',
    '[vite] ready in 438 ms',
    '[vite] local: http://localhost:5173/',
    '[web-admin] GET /api/projects 200 18ms',
  ],
  api: [
    '[api-server] air -c .air.toml',
    '[runner] login shell: /bin/zsh -l -i -c',
    '[api] listening on :8080',
    '[api] GET /health 200 2ms',
  ],
  docs: [
    '[docs-site] npm run docs:dev',
    '[vitepress] serving docs at http://localhost:5174/',
    '[watch] docs/index.md updated',
  ],
  compose: [
    '[compose] docker compose up',
    '[postgres] database system is ready to accept connections',
    '[redis] ready to accept connections',
  ],
}

function cloneProjects(projects) {
  return projects.map((project) => ({
    ...project,
    commands: project.commands.map((command) => ({ ...command })),
    files: [...project.files],
  }))
}

function projectById(projects, projectId) {
  return projects.find((project) => project.id === projectId) || projects[0]
}

export function createHomeDemoState() {
  return {
    projects: cloneProjects(BASE_PROJECTS),
    activeProjectId: 'web',
    activeCommandId: 'dev',
    logLines: [...LOGS_BY_PROJECT.web],
    groupName: 'frontend + api',
  }
}

export function activeHomeDemoProject(state) {
  return projectById(state.projects, state.activeProjectId)
}

export function activeHomeDemoCommand(state) {
  const project = activeHomeDemoProject(state)
  return project.commands.find((command) => command.id === state.activeCommandId) || project.commands[0]
}

export function selectHomeDemoProject(state, projectId) {
  const project = projectById(state.projects, projectId)
  return {
    ...state,
    activeProjectId: project.id,
    activeCommandId: project.commands[0].id,
    logLines: [...LOGS_BY_PROJECT[project.id]],
  }
}

export function setHomeDemoCommand(state, commandId) {
  const project = activeHomeDemoProject(state)
  const command = project.commands.find((item) => item.id === commandId) || project.commands[0]
  return {
    ...state,
    activeCommandId: command.id,
    logLines: [
      ...LOGS_BY_PROJECT[project.id],
      `[preview] ${project.name} -> ${command.command}`,
    ],
  }
}

export function toggleHomeDemoRun(state) {
  const project = activeHomeDemoProject(state)
  const nextStatus = project.status === 'running' ? 'stopped' : 'running'
  const verb = nextStatus === 'running' ? 'started' : 'stopped'
  return {
    ...state,
    projects: state.projects.map((item) => (
      item.id === project.id ? { ...item, status: nextStatus } : item
    )),
    logLines: [
      ...state.logLines,
      `[runner] ${verb} ${project.name}`,
    ],
  }
}

export function toggleHomeDemoGroup(state) {
  const shouldStart = state.projects.some((project) => (
    (project.id === 'web' || project.id === 'api') && project.status !== 'running'
  ))
  const nextStatus = shouldStart ? 'running' : 'stopped'
  const verb = shouldStart ? 'started' : 'stopped'
  return {
    ...state,
    projects: state.projects.map((project) => (
      project.id === 'web' || project.id === 'api' ? { ...project, status: nextStatus } : project
    )),
    logLines: [
      ...state.logLines,
      `[group] ${verb} ${state.groupName}`,
    ],
  }
}

export function simulateHomeDemoScan(state) {
  return {
    ...state,
    logLines: [
      ...state.logLines,
      '[scanner] workspace scan finished: 4 projects, 1 launch group',
    ],
  }
}

export function simulateHomeDemoEdit(state) {
  const project = activeHomeDemoProject(state)
  const command = activeHomeDemoCommand(state)
  return {
    ...state,
    logLines: [
      ...state.logLines,
      `[commands] command editor opened for ${project.name}:${command.label}`,
    ],
  }
}
