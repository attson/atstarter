<script setup>
import { computed, ref } from 'vue'
import AppButton from '../../../../../frontend/src/components/ui/AppButton.vue'
import AppPill from '../../../../../frontend/src/components/ui/AppPill.vue'
import {
  activeHomeDemoCommand,
  activeHomeDemoProject,
  createHomeDemoState,
  selectHomeDemoProject,
  setHomeDemoCommand,
  simulateHomeDemoEdit,
  simulateHomeDemoScan,
  toggleHomeDemoGroup,
  toggleHomeDemoRun,
} from './homeDemoData.mjs'

const state = ref(createHomeDemoState())

const activeProject = computed(() => activeHomeDemoProject(state.value))
const activeCommand = computed(() => activeHomeDemoCommand(state.value))
const runningCount = computed(() => (
  state.value.projects.filter((project) => project.status === 'running').length
))
const exitedCount = computed(() => (
  state.value.projects.filter((project) => project.status !== 'running').length
))
const selectedStatus = computed(() => ({ State: activeProject.value.status }))
const activeCommandLine = computed(() => activeCommand.value.command)

function statusVariant(status) {
  return status === 'running' ? 'running' : 'stopped'
}

function selectProject(projectId) {
  state.value = selectHomeDemoProject(state.value, projectId)
}

function selectCommand(event) {
  state.value = setHomeDemoCommand(state.value, event.target.value)
}

function toggleRun() {
  state.value = toggleHomeDemoRun(state.value)
}

function toggleGroup() {
  state.value = toggleHomeDemoGroup(state.value)
}

function scanWorkspace() {
  state.value = simulateHomeDemoScan(state.value)
}

function editCommand() {
  state.value = simulateHomeDemoEdit(state.value)
}
</script>

<template>
  <section class="home-demo reveal" aria-label="AT Starter mock workspace">
    <div class="app-shell app-shell-demo">
      <header class="topbar">
        <div class="brand">atstarter</div>
        <div class="summary">
          <span class="summary-count">{{ state.projects.length }} projects</span>
          <AppPill variant="running" dot>{{ runningCount }} running</AppPill>
          <AppPill variant="exited">{{ exitedCount }} exited</AppPill>
        </div>
        <div class="tabs">
          <button class="tab active" type="button">Projects</button>
          <button class="tab" type="button">Containers</button>
        </div>
        <div class="top-actions">
          <AppButton variant="secondary" size="sm">New Group</AppButton>
          <AppButton variant="secondary" size="sm" @click="scanWorkspace">Scan</AppButton>
          <AppButton variant="primary" size="sm">Add</AppButton>
        </div>
      </header>

      <main class="workspace">
        <aside class="project-list">
          <div class="search-wrap">
            <div class="search-field">
              <input class="search" value="" placeholder="Search projects, path, command..." readonly />
            </div>
          </div>

          <div class="tree-scroll">
            <div class="group-section">
              <div class="section-title">Groups</div>
              <button
                type="button"
                class="group-row"
                @click="toggleGroup"
              >
                <span class="project-main">
                  <span class="status-dot running"></span>
                  <span class="project-name">{{ state.groupName }}</span>
                  <span class="type-pill">2</span>
                </span>
              </button>
            </div>

            <button
              v-for="project in state.projects"
              :key="project.id"
              type="button"
              :class="['tree-row', 'project-row', { active: project.id === state.activeProjectId }]"
              :aria-label="`${project.name}, ${project.kind}, ${project.status}`"
              @click="selectProject(project.id)"
            >
              <span class="project-spacer"></span>
              <span class="project-main">
                <span :class="['status-dot', statusVariant(project.status)]"></span>
                <span class="project-name">{{ project.name }}</span>
                <span class="type-pill">{{ project.kind }}</span>
              </span>
            </button>
          </div>
        </aside>

        <section class="detail">
          <div class="project-header">
            <div class="info">
              <div class="title-line">
                <h1>{{ activeProject.name }}</h1>
                <AppPill :variant="statusVariant(activeProject.status)" :dot="activeProject.status === 'running'">
                  {{ activeProject.status }}
                </AppPill>
                <AppPill variant="neutral">{{ activeProject.kind }}</AppPill>
                <AppPill variant="neutral" class="branch-pill">{{ activeProject.branch }}</AppPill>
              </div>
              <div class="path" :title="activeProject.path">{{ activeProject.path }}</div>
              <div class="command-box">
                <span class="cmd-label">CMD</span>
                <div class="command-picker">
                  <select class="command-trigger" :value="state.activeCommandId" @change="selectCommand">
                    <option
                      v-for="command in activeProject.commands"
                      :key="command.id"
                      :value="command.id"
                    >
                      {{ command.label }}
                    </option>
                  </select>
                </div>
                <code>{{ activeCommandLine }}</code>
                <AppButton class="command-edit" variant="secondary" size="sm" @click="editCommand">Edit</AppButton>
              </div>
            </div>
            <div class="btns">
              <AppButton variant="secondary" size="sm">Add Group</AppButton>
              <div class="run-controls">
                <AppButton
                  variant="danger"
                  size="sm"
                  :disabled="activeProject.status !== 'running'"
                  @click="toggleRun"
                >
                  Stop
                </AppButton>
                <AppButton
                  variant="secondary"
                  size="sm"
                  :disabled="activeProject.status !== 'running'"
                  @click="toggleRun"
                >
                  Restart
                </AppButton>
                <AppButton
                  variant="success"
                  size="sm"
                  :disabled="activeProject.status === 'running'"
                  @click="toggleRun"
                >
                  Start
                </AppButton>
              </div>
            </div>
          </div>

          <div class="detail-tabs">
            <button class="tab active" type="button">日志</button>
            <button class="tab" type="button">文件</button>
          </div>
          <div class="detail-tab-body">
            <div class="log-wrap">
              <div :class="['banner', statusVariant(selectedStatus.State)]">
                {{ selectedStatus.State === 'running' ? '● 运行中' : '○ 未运行' }}
              </div>
              <div class="term-area">
                <pre class="term-host"><span v-for="line, index in state.logLines" :key="`${index}-${line}`">{{ line }}
</span></pre>
              </div>
            </div>
          </div>
        </section>
      </main>
    </div>
  </section>
</template>

<style scoped>
.home-demo {
  position: relative;
  z-index: 1;
  width: min(1180px, calc(100vw - 48px));
  margin: 20px auto 0;

  --space-1: 2px;
  --space-2: 4px;
  --space-3: 6px;
  --space-4: 8px;
  --space-5: 10px;
  --space-6: 12px;
  --space-7: 16px;
  --space-8: 20px;
  --space-9: 24px;
  --space-10: 32px;
  --radius-sm: 5px;
  --radius-md: 8px;
  --radius-lg: 12px;
  --radius-full: 999px;
  --shadow-sm: 0 1px 2px rgba(0, 0, 0, .04);
  --shadow-md: 0 8px 24px rgba(0, 0, 0, .10);
  --shadow-lg: 0 20px 40px rgba(0, 0, 0, .22);
  --dur-fast: 120ms;
  --dur-base: 200ms;
  --dur-slow: 260ms;
  --ease: cubic-bezier(.2, 0, 0, 1);
  --ease-spring: cubic-bezier(.34, 1.35, .64, 1);
  --font-sans:
    -apple-system, BlinkMacSystemFont, "SF Pro Text", "Segoe UI",
    "PingFang SC", "Microsoft YaHei", "Helvetica Neue", sans-serif;
  --font-mono:
    "SFMono-Regular", ui-monospace, Consolas, "Liberation Mono", monospace;
  --fs-lg: 22px;
  --fs-md: 16px;
  --fs-base: 13px;
  --fs-sm: 12px;
  --fs-xs: 11px;
  --fs-mono: 12px;
  --fw-regular: 400;
  --fw-medium: 500;
  --fw-semibold: 600;
}

.home-demo,
.home-demo * {
  box-sizing: border-box;
}

.app-shell-demo {
  grid-template-rows: 48px minmax(0, 1fr);
  height: 620px;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--bg-gradient);
  box-shadow: var(--shadow-lg);
}

.app-shell-demo > .topbar { grid-row: 1; }
.app-shell-demo > .workspace { grid-row: 2; }

.app-shell {
  display: grid;
  font-family: var(--font-sans);
  color: var(--text);
}

.topbar {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: var(--space-7);
  padding: 0 var(--space-7);
  background: linear-gradient(180deg, rgba(255, 255, 255, .02), transparent);
  border-bottom: 1px solid var(--border);
  box-shadow: var(--surface-highlight);
}

.brand {
  font-size: 17px;
  font-weight: var(--fw-semibold);
  letter-spacing: 0;
  background: var(--brand-gradient);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
  color: var(--text);
}

.summary {
  display: flex;
  align-items: center;
  min-width: 0;
  flex-wrap: wrap;
  gap: var(--space-4);
  color: var(--text-muted);
  font-size: var(--fs-sm);
}

.summary-count {
  color: var(--text-secondary);
  font-weight: var(--fw-medium);
  white-space: nowrap;
}

.tabs {
  display: flex;
  gap: var(--space-2);
}

.tab {
  height: 28px;
  padding: 0 var(--space-5);
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  font: inherit;
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  cursor: pointer;
}

.tab.active {
  background: var(--elevated);
  color: var(--text);
  border-color: var(--border-strong);
}

.top-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
  flex-shrink: 0;
  gap: var(--space-4);
}

.workspace {
  min-height: 0;
  display: flex;
}

.project-list {
  width: 300px;
  min-width: 280px;
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  background: linear-gradient(180deg, rgba(255, 255, 255, .015), transparent), var(--surface);
  box-shadow: var(--surface-highlight);
  min-height: 0;
}

.search-wrap {
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--border);
}

.search-field {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  align-items: center;
}

.search {
  width: 100%;
  height: 30px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--elevated-gradient);
  color: var(--text);
  padding: 0 var(--space-4);
  font: inherit;
  font-size: var(--fs-sm);
  outline: none;
  box-shadow: var(--surface-highlight);
}

.tree-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: var(--space-3) var(--space-4);
}

.group-section {
  margin-bottom: var(--space-4);
  padding-bottom: var(--space-3);
  border-bottom: 1px solid var(--border);
}

.section-title {
  color: var(--text-subtle);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  letter-spacing: 0.03em;
  text-transform: uppercase;
  margin: var(--space-2) var(--space-2) var(--space-3);
}

.tree-row,
.group-row {
  width: 100%;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.project-row,
.group-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-height: 28px;
  margin: 1px 0;
  padding: 2px var(--space-4) 2px 4px;
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
}

.project-row:hover,
.group-row:hover {
  background: var(--elevated-gradient);
}

.project-row.active {
  background: var(--elevated-gradient);
  color: var(--text);
  box-shadow: inset 0 0 0 1px var(--border-strong), var(--surface-highlight);
}

.project-spacer {
  width: 12px;
  flex: 0 0 auto;
}

.project-main {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: var(--space-3);
  color: inherit;
  font: inherit;
}

.project-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.branch-pill {
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.type-pill {
  flex: 0 0 auto;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-full);
  color: var(--text-muted);
  background: transparent;
  padding: 1px 7px;
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex: 0 0 auto;
}

.status-dot.running {
  background: var(--accent-strong);
  animation: pulse-ring 2s ease-in-out infinite;
}

.status-dot.stopped {
  background: var(--text-subtle);
}

.detail {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: transparent;
}

.project-header {
  display: grid;
  grid-template-columns: minmax(280px, 1fr) auto;
  align-items: start;
  column-gap: var(--space-8);
  row-gap: var(--space-3);
  padding: var(--space-6) var(--space-8);
  border-bottom: 1px solid var(--border);
  background: transparent;
}

.info {
  min-width: 0;
  max-width: 100%;
  display: contents;
}

.title-line {
  grid-column: 1;
  grid-row: 1;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-4);
  min-width: 0;
}

h1 {
  max-width: min(560px, 100%);
  margin: 0;
  color: var(--text);
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  letter-spacing: 0;
  line-height: 1.15;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.path {
  grid-column: 1 / -1;
  max-width: 100%;
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  line-height: 1.25;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.command-box {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  gap: var(--space-4);
  width: min(100%, 760px);
  min-width: 0;
  padding: var(--space-3) var(--space-5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--elevated-gradient);
  box-shadow: var(--surface-highlight);
}

.cmd-label {
  color: var(--text-subtle);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  letter-spacing: 0.03em;
}

.command-picker {
  position: relative;
  flex-shrink: 0;
}

.command-trigger {
  height: 24px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--elevated-gradient);
  color: var(--text);
  padding: 0 var(--space-3);
  font: inherit;
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
}

.command-box code {
  flex: 1;
  min-width: 0;
  color: var(--text);
  font-family: var(--font-mono);
  font-size: var(--fs-mono);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.command-edit {
  margin-left: auto;
  flex-shrink: 0;
}

.btns {
  grid-column: 2;
  grid-row: 1;
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: var(--space-3);
  flex-wrap: wrap;
  row-gap: var(--space-3);
}

.run-controls {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  margin-left: var(--space-3);
  padding-left: var(--space-4);
  border-left: 1px solid var(--border);
}

.detail-tabs {
  display: flex;
  gap: var(--space-2);
  padding: var(--space-4) var(--space-8) 0;
}

.detail-tab-body {
  flex: 1;
  min-height: 0;
  display: flex;
}

.log-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: var(--log-bg);
  position: relative;
}

.log-wrap::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 1px;
  background: var(--log-hairline);
  pointer-events: none;
  z-index: 2;
}

.banner {
  height: 34px;
  display: flex;
  align-items: center;
  padding: 0 var(--space-6);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  letter-spacing: 0.03em;
  border-bottom: 1px solid var(--log-border);
  box-shadow: var(--log-highlight);
}

.banner.running {
  color: var(--log-banner-running);
  background: var(--log-banner-running-bg);
}

.banner.stopped {
  color: var(--log-banner-stopped);
  background: var(--log-banner-stopped-bg);
}

.term-area {
  position: relative;
  flex: 1;
  min-height: 0;
}

.term-host {
  width: 100%;
  height: 100%;
  min-height: 0;
  margin: 0;
  padding: var(--space-7);
  overflow: auto;
  background: transparent;
  color: var(--log-text);
  font-family: var(--font-mono);
  font-size: 11px;
  line-height: 1.55;
}

@keyframes pulse-ring {
  0%, 100% {
    box-shadow:
      0 0 0 2.5px var(--success-soft),
      0 0 6px var(--success-glow-a);
  }
  50% {
    box-shadow:
      0 0 0 5px var(--success-soft),
      0 0 10px var(--success-glow-b);
  }
}

@media (max-width: 980px) {
  .home-demo {
    width: calc(100vw - 48px);
  }

  .app-shell-demo {
    grid-template-rows: auto minmax(0, 1fr);
    height: auto;
    min-height: 760px;
  }

  .topbar {
    height: auto;
    min-height: 48px;
    flex-wrap: wrap;
    gap: var(--space-4);
    padding: var(--space-5);
  }

  .top-actions {
    width: 100%;
    margin-left: 0;
    overflow: auto;
  }

  .workspace {
    flex-direction: column;
  }

  .project-list {
    width: 100%;
    min-width: 0;
    max-height: 260px;
    border-right: 0;
    border-bottom: 1px solid var(--border);
  }

  .project-header {
    grid-template-columns: 1fr;
    padding: var(--space-5);
  }

  .title-line {
    align-items: flex-start;
  }

  .branch-pill {
    max-width: 130px;
  }

  .btns {
    grid-column: 1;
    grid-row: auto;
    justify-content: flex-start;
  }

  .command-box {
    flex-wrap: wrap;
  }

  .command-box code {
    flex-basis: 100%;
  }
}
</style>
