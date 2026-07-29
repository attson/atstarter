<script setup>
import { computed, ref } from 'vue'
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
  <section class="home-demo reveal" aria-label="AT Starter interactive demo">
    <div class="demo-heading">
      <div>
        <p class="demo-kicker">Live mock workspace</p>
        <h2>首页直接体验项目启动器</h2>
      </div>
      <div class="demo-version">
        <span class="pulse"></span>
        <span>{{ runningCount }} running</span>
      </div>
    </div>

    <div class="launcher">
      <aside class="project-pane">
        <div class="pane-top">
          <div>
            <strong>Projects</strong>
            <span>~/code/acme</span>
          </div>
          <button type="button" title="Scan workspace" @click="scanWorkspace">Scan</button>
        </div>

        <button
          v-for="project in state.projects"
          :key="project.id"
          type="button"
          class="project-row"
          :class="{ active: project.id === state.activeProjectId }"
          :aria-label="`${project.name}, ${project.kind}, ${project.status}`"
          @click="selectProject(project.id)"
        >
          <span class="project-main">
            <span class="project-name">{{ project.name }}</span>
            <span class="project-meta">{{ project.kind }} · {{ project.branch }}</span>
          </span>
          <span class="status-dot" :class="project.status"></span>
        </button>

        <div class="group-card">
          <div>
            <strong>{{ state.groupName }}</strong>
            <span>web-admin + api-server</span>
          </div>
          <button type="button" @click="toggleGroup">
            Toggle
          </button>
        </div>
      </aside>

      <main class="detail-pane">
        <div class="detail-header">
          <div class="title-block">
            <div class="title-line">
              <h3>{{ activeProject.name }}</h3>
              <span class="type-pill">{{ activeProject.kind }}</span>
              <span class="status-pill" :class="activeProject.status">{{ activeProject.status }}</span>
            </div>
            <p :title="activeProject.path">{{ activeProject.path }}</p>
          </div>
          <button type="button" class="run-button" @click="toggleRun">
            {{ activeProject.status === 'running' ? 'Stop' : 'Start' }}
          </button>
        </div>

        <div class="command-row">
          <select :value="state.activeCommandId" @change="selectCommand">
            <option
              v-for="command in activeProject.commands"
              :key="command.id"
              :value="command.id"
            >
              {{ command.label }}
            </option>
          </select>
          <code>{{ activeCommand.command }}</code>
          <button type="button" title="Edit command" @click="editCommand">Edit</button>
        </div>

        <div class="workspace-grid">
          <section class="console-panel">
            <div class="panel-title">
              <span>Logs</span>
              <span>stdout / stderr</span>
            </div>
            <pre><span v-for="line, index in state.logLines" :key="`${index}-${line}`">{{ line }}
</span></pre>
          </section>

          <section class="files-panel">
            <div class="panel-title">
              <span>Files</span>
              <span>{{ activeProject.files.length }} items</span>
            </div>
            <ul>
              <li v-for="file in activeProject.files" :key="file">
                <span class="file-icon"></span>
                <span>{{ file }}</span>
              </li>
            </ul>
          </section>
        </div>
      </main>
    </div>
  </section>
</template>

<style scoped>
.home-demo {
  position: relative;
  z-index: 1;
  box-sizing: border-box;
  width: 100%;
  max-width: 1180px;
  margin: 30px auto 0;
  padding: 0 24px;
}

.home-demo *,
.home-demo *::before,
.home-demo *::after {
  box-sizing: border-box;
}

.demo-heading {
  display: flex;
  align-items: end;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 18px;
}

.demo-kicker {
  margin: 0 0 6px;
  color: var(--home-text-muted);
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.demo-heading h2 {
  margin: 0;
  border: 0;
  padding: 0;
  color: var(--home-text-strong);
  font-size: 28px;
  font-weight: 800;
  letter-spacing: 0;
  line-height: 1.2;
}

.demo-version {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 34px;
  padding: 0 12px;
  border: 1px solid var(--home-border);
  border-radius: 999px;
  background: var(--home-surface);
  color: var(--home-text);
  font-size: 13px;
  font-weight: 700;
  white-space: nowrap;
}

.pulse {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #22c55e;
  box-shadow: 0 0 0 5px color-mix(in srgb, #22c55e 18%, transparent);
}

.launcher {
  display: grid;
  grid-template-columns: minmax(240px, 290px) minmax(0, 1fr);
  width: 100%;
  max-width: 100%;
  min-height: 560px;
  overflow: hidden;
  border: 1px solid var(--home-border);
  border-radius: 14px;
  background: color-mix(in srgb, var(--home-bg-2) 86%, #020617);
  box-shadow:
    0 24px 70px var(--home-frame-shadow),
    0 0 0 1px color-mix(in srgb, var(--home-indigo) 10%, transparent);
}

.project-pane {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
  max-width: 100%;
  padding: 14px;
  border-right: 1px solid var(--home-border);
  background: color-mix(in srgb, var(--home-surface-2) 72%, var(--home-bg));
}

.pane-top,
.group-card,
.detail-header,
.command-row,
.panel-title {
  display: flex;
  align-items: center;
}

.pane-top {
  justify-content: space-between;
  gap: 12px;
  padding: 2px 2px 10px;
}

.pane-top div,
.group-card div,
.title-block {
  min-width: 0;
}

.pane-top strong,
.group-card strong {
  display: block;
  color: var(--home-text-strong);
  font-size: 14px;
}

.pane-top span,
.group-card span {
  display: block;
  overflow: hidden;
  color: var(--home-text-muted);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

button,
select {
  border: 1px solid var(--home-border);
  border-radius: 8px;
  background: var(--home-surface-2);
  color: var(--home-text);
  font: inherit;
}

button {
  min-height: 32px;
  padding: 0 12px;
  cursor: pointer;
}

button:hover,
select:hover {
  border-color: color-mix(in srgb, var(--home-indigo) 48%, transparent);
}

.project-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
  min-height: 58px;
  padding: 10px 12px;
  text-align: left;
}

.project-row.active {
  border-color: color-mix(in srgb, var(--home-indigo) 70%, transparent);
  background: color-mix(in srgb, var(--home-indigo) 14%, var(--home-surface-2));
}

.project-main {
  min-width: 0;
}

.project-name,
.project-meta {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-name {
  color: var(--home-text-strong);
  font-size: 14px;
  font-weight: 750;
}

.project-meta {
  margin-top: 3px;
  color: var(--home-text-muted);
  font-size: 12px;
}

.status-dot {
  width: 9px;
  height: 9px;
  flex: 0 0 auto;
  border-radius: 50%;
  background: #94a3b8;
}

.status-dot.running {
  background: #22c55e;
  box-shadow: 0 0 0 4px color-mix(in srgb, #22c55e 16%, transparent);
}

.status-dot.stopped {
  background: #f59e0b;
}

.group-card {
  justify-content: space-between;
  gap: 12px;
  margin-top: auto;
  padding: 12px;
  border: 1px solid var(--home-border);
  border-radius: 10px;
  background: color-mix(in srgb, var(--home-cyan) 9%, var(--home-surface));
}

.detail-pane {
  min-width: 0;
  max-width: 100%;
  padding: 18px;
}

.detail-header {
  justify-content: space-between;
  gap: 18px;
}

.title-line {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.title-line h3 {
  overflow: hidden;
  margin: 0;
  color: var(--home-text-strong);
  font-size: 22px;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.title-block p {
  overflow: hidden;
  margin: 6px 0 0;
  color: var(--home-text-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.type-pill,
.status-pill {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 0 8px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 750;
  white-space: nowrap;
}

.type-pill {
  border: 1px solid var(--home-border);
  color: var(--home-text);
}

.status-pill.running {
  background: color-mix(in srgb, #22c55e 16%, transparent);
  color: #047857;
}

.status-pill.stopped {
  background: color-mix(in srgb, #f59e0b 16%, transparent);
  color: #92400e;
}

:global(.dark) .status-pill.running {
  color: #86efac;
}

:global(.dark) .status-pill.stopped {
  color: #fbbf24;
}

.run-button {
  min-width: 86px;
  border: 0;
  background: linear-gradient(120deg, var(--home-indigo), var(--home-violet));
  color: #fff;
  font-weight: 800;
}

.command-row {
  gap: 10px;
  margin-top: 18px;
  padding: 10px;
  border: 1px solid var(--home-border);
  border-radius: 10px;
  background: var(--home-surface);
}

.command-row select {
  min-width: 96px;
  height: 34px;
  padding: 0 9px;
}

.command-row code {
  display: block;
  min-width: 0;
  flex: 1;
  overflow: hidden;
  border-radius: 7px;
  background: color-mix(in srgb, #020617 88%, var(--home-bg));
  color: #c4f1ff;
  font-size: 12px;
  line-height: 34px;
  padding: 0 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workspace-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.35fr) minmax(220px, 0.65fr);
  gap: 14px;
  margin-top: 14px;
}

.console-panel,
.files-panel {
  min-width: 0;
  border: 1px solid var(--home-border);
  border-radius: 10px;
  background: var(--home-surface);
  overflow: hidden;
}

.panel-title {
  justify-content: space-between;
  gap: 12px;
  min-height: 42px;
  padding: 0 14px;
  border-bottom: 1px solid var(--home-border);
  color: var(--home-text-strong);
  font-size: 13px;
  font-weight: 800;
}

.panel-title span:last-child {
  color: var(--home-text-muted);
  font-size: 12px;
  font-weight: 650;
}

pre {
  min-height: 300px;
  margin: 0;
  padding: 16px;
  overflow: auto;
  background: color-mix(in srgb, #020617 92%, var(--home-bg));
  color: #d1fae5;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  line-height: 1.9;
}

ul {
  display: grid;
  gap: 6px;
  margin: 0;
  padding: 12px;
  list-style: none;
}

li {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  min-height: 34px;
  padding: 0 8px;
  border-radius: 8px;
  color: var(--home-text);
  font-size: 12px;
}

li:hover {
  background: color-mix(in srgb, var(--home-indigo) 10%, transparent);
}

li span:last-child {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-icon {
  width: 13px;
  height: 16px;
  flex: 0 0 auto;
  border: 1px solid color-mix(in srgb, var(--home-cyan) 64%, transparent);
  border-radius: 3px;
  background: color-mix(in srgb, var(--home-cyan) 14%, transparent);
}

@media (max-width: 900px) {
  .launcher {
    grid-template-columns: 1fr;
  }

  .project-pane {
    border-right: 0;
    border-bottom: 1px solid var(--home-border);
  }

  .workspace-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .home-demo {
    left: auto;
    width: calc(100vw - 48px);
    max-width: calc(100vw - 48px);
    margin-top: 24px;
    margin-left: 0;
    margin-right: 0;
    padding: 0;
  }

  .demo-heading,
  .detail-header,
  .command-row {
    align-items: stretch;
    flex-direction: column;
  }

  .demo-heading h2 {
    font-size: 24px;
  }

  .launcher {
    width: 100%;
    min-height: 0;
    border-radius: 12px;
  }

  .project-pane {
    padding: 14px;
  }

  .group-card {
    align-items: stretch;
    flex-direction: column;
  }

  .group-card button {
    width: 100%;
  }

  .detail-pane {
    padding: 14px;
  }

  .title-line {
    flex-wrap: wrap;
  }

  .command-row code {
    width: 100%;
    flex: none;
  }

  pre {
    min-height: 240px;
  }
}
</style>
