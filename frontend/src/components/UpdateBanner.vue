<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Download, X, Loader2, Sparkles, AlertTriangle } from 'lucide-vue-next'
import { isUpdateBannerVisible, startUpdateCheckTimer } from '../updateSchedule'
import {
  UpdateGetState,
  UpdateCheck,
  UpdateStartDownload,
  UpdateCancel,
  UpdateInstall,
} from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import AppButton from './ui/AppButton.vue'
import AppIcon from './ui/AppIcon.vue'

const state = ref({
  current: '',
  latest: '',
  available: false,
  notes: '',
  checking: false,
  downloading: false,
  downloadPct: 0,
  ready: false,
  error: '',
  assetUrl: '',
  assetSize: 0,
  canInstall: false,
  lastCheckAt: 0,
})
const dismissed = ref(false)
const manualNotice = ref(false)

const visible = computed(() => {
  return isUpdateBannerVisible(state.value, {
    dismissed: dismissed.value,
    manualNotice: manualNotice.value,
  })
})

const kind = computed(() => {
  const s = state.value
  if (s.checking) return 'checking'
  if (s.error) return 'error'
  if (s.ready) return 'ready'
  if (s.downloading) return 'downloading'
  if (s.available) return 'available'
  return 'idle'
})

async function refresh() {
  try { state.value = await UpdateGetState() } catch {}
}

async function check({ notify = false } = {}) {
  if (notify) manualNotice.value = true
  dismissed.value = false
  try {
    state.value = await UpdateCheck()
  } catch (e) {
    state.value = { ...state.value, checking: false, error: String(e) }
  }
}

async function download() {
  dismissed.value = false
  state.value = await UpdateStartDownload()
}

async function cancel() {
  state.value = await UpdateCancel()
}

async function install() {
  state.value = await UpdateInstall()
}

function dismiss() {
  dismissed.value = true
  manualNotice.value = false
}

function openReleasePage() {
  if (state.value.assetUrl) {
    // Fall back to the release detail page (strip the asset filename).
    const url = state.value.assetUrl.replace(/\/[^/]+$/, '')
    window.open(url, '_blank')
  }
}

let unsub = null
let stopTimer = null
onMounted(() => {
  refresh()
  unsub = EventsOn('update:state', (s) => { state.value = s })
  // Silent checks only surface when an update is available.
  check()
  stopTimer = startUpdateCheckTimer(() => check())
})
onUnmounted(() => {
  if (unsub) EventsOff('update:state')
  if (stopTimer) stopTimer()
})

defineExpose({ check })
</script>

<template>
  <transition name="dlg-fade">
    <div v-if="visible" :class="['update-banner', `kind-${kind}`]">
      <div class="icon-wrap">
        <AppIcon v-if="kind === 'downloading'" :icon="Loader2" :size="16" class="spin" />
        <AppIcon v-else-if="kind === 'error'" :icon="AlertTriangle" :size="16" />
        <AppIcon v-else :icon="Sparkles" :size="16" />
      </div>

      <div class="body">
        <template v-if="kind === 'error'">
          <div class="title">更新出错</div>
          <div class="sub">{{ state.error }}</div>
        </template>
        <template v-else-if="kind === 'checking'">
          <div class="title">正在检查更新…</div>
          <div class="sub">当前 {{ state.current || 'dev' }}</div>
        </template>
        <template v-else-if="kind === 'ready'">
          <div class="title">{{ state.latest }} 已下载完成</div>
          <div class="sub">点击「立即安装」重启应用并升级。</div>
        </template>
        <template v-else-if="kind === 'downloading'">
          <div class="title">下载 {{ state.latest }}… {{ state.downloadPct }}%</div>
          <div class="progress"><div class="bar" :style="{ width: state.downloadPct + '%' }" /></div>
        </template>
        <template v-else-if="kind === 'available'">
          <div class="title">新版本 {{ state.latest }} 可用</div>
          <div class="sub">当前 {{ state.current }} · {{ (state.assetSize / 1024 / 1024).toFixed(1) }} MB</div>
        </template>
        <template v-else>
          <div class="title">已是最新版本</div>
          <div class="sub">当前 {{ state.current || state.latest || 'dev' }}</div>
        </template>
      </div>

      <div class="actions">
        <template v-if="kind === 'checking'">
          <AppButton variant="secondary" size="sm" icon-only @click="dismiss" aria-label="dismiss">
            <template #icon><AppIcon :icon="X" :size="12" /></template>
          </AppButton>
        </template>
        <template v-else-if="kind === 'available'">
          <AppButton v-if="state.canInstall" variant="primary" size="sm" @click="download">
            <template #icon><AppIcon :icon="Download" :size="12" /></template>
            下载
          </AppButton>
          <AppButton v-else variant="secondary" size="sm" @click="openReleasePage">前往下载</AppButton>
          <AppButton variant="secondary" size="sm" icon-only @click="dismiss" aria-label="dismiss">
            <template #icon><AppIcon :icon="X" :size="12" /></template>
          </AppButton>
        </template>
        <template v-else-if="kind === 'downloading'">
          <AppButton variant="secondary" size="sm" @click="cancel">取消</AppButton>
        </template>
        <template v-else-if="kind === 'ready'">
          <AppButton variant="primary" size="sm" @click="install">立即安装</AppButton>
          <AppButton variant="secondary" size="sm" icon-only @click="dismiss" aria-label="dismiss">
            <template #icon><AppIcon :icon="X" :size="12" /></template>
          </AppButton>
        </template>
        <template v-else-if="kind === 'error'">
          <AppButton variant="secondary" size="sm" @click="check">重试</AppButton>
          <AppButton variant="secondary" size="sm" icon-only @click="dismiss" aria-label="dismiss">
            <template #icon><AppIcon :icon="X" :size="12" /></template>
          </AppButton>
        </template>
      </div>
    </div>
  </transition>
</template>

<style scoped>
.update-banner {
  display: flex;
  align-items: center;
  gap: var(--space-5);
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--border);
  background: var(--elevated-gradient);
  box-shadow: var(--surface-highlight);
  font-size: var(--fs-sm);
}

.update-banner.kind-available { border-bottom-color: var(--success-line); background: var(--success-gradient); }
.update-banner.kind-downloading { border-bottom-color: var(--border-strong); }
.update-banner.kind-ready { border-bottom-color: var(--success-line); background: var(--success-gradient); }
.update-banner.kind-error { border-bottom-color: var(--danger-line); background: var(--danger-gradient); color: var(--danger-fg); }

.icon-wrap {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: var(--radius-md);
  background: rgba(255, 255, 255, .06);
  color: var(--accent-strong);
  flex: 0 0 auto;
  box-shadow: var(--surface-highlight);
}
.update-banner.kind-error .icon-wrap { color: var(--danger-fg); }

.body {
  min-width: 0;
  flex: 1 1 auto;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.title {
  color: var(--text);
  font-weight: var(--fw-semibold);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.update-banner.kind-error .title { color: var(--danger-fg); }
.sub {
  color: var(--text-muted);
  font-size: var(--fs-xs);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.progress {
  margin-top: var(--space-2);
  height: 4px;
  border-radius: 2px;
  background: rgba(255, 255, 255, .06);
  overflow: hidden;
}
.bar {
  height: 100%;
  background: var(--primary-gradient);
  transition: width var(--dur-fast) linear;
}

.actions {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  flex: 0 0 auto;
}

.spin {
  animation: update-spin 0.9s linear infinite;
}
@keyframes update-spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
