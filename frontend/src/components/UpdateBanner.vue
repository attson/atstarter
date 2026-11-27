<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
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

const { t } = useI18n()

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
const failure = ref(null)
const installing = ref(false)
const errorMessage = computed(() => failure.value
  ? failure.value.detail || t('updates.failure')
  : state.value.error
)
const currentVersion = computed(() => state.value.current || state.value.latest || 'dev')
const checkingVersion = computed(() => state.value.current || 'dev')

const visible = computed(() => {
  return isUpdateBannerVisible(failure.value ? { ...state.value, error: true } : state.value, {
    dismissed: dismissed.value,
    manualNotice: manualNotice.value,
  })
})

const kind = computed(() => {
  const s = state.value
  if (s.checking) return 'checking'
  if (failure.value || s.error) return 'error'
  if (installing.value) return 'installing'
  if (s.ready) return 'ready'
  // The existing backend keeps downloading true through signature/checksum
  // verification after reporting 100%; unknown-size downloads may stay below it.
  if (s.downloading && s.downloadPct >= 100) return 'verifying'
  if (s.downloading) return 'downloading'
  if (s.available) return 'available'
  return 'idle'
})

async function refresh() {
  try { state.value = await UpdateGetState() } catch {}
}

async function runUpdate(action) {
  failure.value = null
  try {
    state.value = await action()
  } catch (error) {
    // Wails may reject with a raw string or a message-shaped native error.
    // Keep detail byte-for-byte; only an absent detail receives a translated fallback.
    const detail = typeof error === 'string' ? error : typeof error?.message === 'string' ? error.message : ''
    failure.value = { detail }
    state.value = { ...state.value, checking: false, downloading: false }
  }
}

async function check({ notify = false } = {}) {
  if (notify) manualNotice.value = true
  dismissed.value = false
  await runUpdate(UpdateCheck)
}

async function download() {
  dismissed.value = false
  await runUpdate(UpdateStartDownload)
}

async function cancel() {
  await runUpdate(UpdateCancel)
}

async function install() {
  installing.value = true
  try { await runUpdate(UpdateInstall) }
  finally { installing.value = false }
}

let autoHideTimer = null
function clearAutoHide() {
  if (autoHideTimer) { clearTimeout(autoHideTimer); autoHideTimer = null }
}

function dismiss() {
  clearAutoHide()
  dismissed.value = true
  manualNotice.value = false
}

// After a manual "already up to date" result, fade the banner out on its own
// so the user isn't forced to close a purely informational notice by hand.
watch([kind, manualNotice], ([k, manual]) => {
  clearAutoHide()
  if (manual && k === 'idle') {
    autoHideTimer = setTimeout(dismiss, 4000)
  }
})

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
  unsub = EventsOn('update:state', (s) => { failure.value = null; state.value = s })
  // Silent checks only surface when an update is available.
  check()
  stopTimer = startUpdateCheckTimer(() => check())
})
onUnmounted(() => {
  if (unsub) EventsOff('update:state')
  if (stopTimer) stopTimer()
  clearAutoHide()
})

defineExpose({ check })
</script>

<template>
  <transition name="dlg-fade">
    <div v-if="visible" :class="['update-banner', `kind-${kind}`]">
      <div class="icon-wrap">
        <AppIcon v-if="kind === 'downloading' || kind === 'verifying' || kind === 'installing'" :icon="Loader2" :size="16" class="spin" />
        <AppIcon v-else-if="kind === 'error'" :icon="AlertTriangle" :size="16" />
        <AppIcon v-else :icon="Sparkles" :size="16" />
      </div>

      <div class="body">
        <template v-if="kind === 'error'">
          <div class="title">{{ t('updates.error') }}</div>
          <div class="sub">{{ errorMessage }}</div>
        </template>
        <template v-else-if="kind === 'checking'">
          <div class="title">{{ t('updates.checking') }}</div>
          <div class="sub">{{ t('updates.current', { version: checkingVersion }) }}</div>
        </template>
        <template v-else-if="kind === 'installing'">
          <div class="title">{{ t('updates.installing', { version: state.latest }) }}</div>
        </template>
        <template v-else-if="kind === 'ready'">
          <div class="title">{{ t('updates.ready', { version: state.latest }) }}</div>
          <div class="sub">{{ t('updates.installHint') }}</div>
        </template>
        <template v-else-if="kind === 'downloading' || kind === 'verifying'">
          <div class="title">{{ kind === 'verifying' ? t('updates.verifying', { version: state.latest }) : t('updates.downloading', { version: state.latest, percent: state.downloadPct }) }}</div>
          <div class="progress"><div class="bar" :style="{ width: state.downloadPct + '%' }" /></div>
        </template>
        <template v-else-if="kind === 'available'">
          <div class="title">{{ t('updates.available', { version: state.latest }) }}</div>
          <div class="sub">{{ t('updates.currentSize', { version: state.current, size: (state.assetSize / 1024 / 1024).toFixed(1) }) }}</div>
        </template>
        <template v-else>
          <div class="title">{{ t('updates.upToDate') }}</div>
          <div class="sub">{{ t('updates.current', { version: currentVersion }) }}</div>
        </template>
      </div>

      <div class="actions">
        <template v-if="kind === 'checking'">
          <AppButton variant="secondary" size="sm" icon-only @click="dismiss" :aria-label="t('updates.dismiss')">
            <template #icon><AppIcon :icon="X" :size="12" /></template>
          </AppButton>
        </template>
        <template v-else-if="kind === 'available'">
          <AppButton v-if="state.canInstall" variant="primary" size="sm" @click="download">
            <template #icon><AppIcon :icon="Download" :size="12" /></template>
            {{ t('updates.download') }}
          </AppButton>
          <AppButton v-else variant="secondary" size="sm" @click="openReleasePage">{{ t('updates.openDownload') }}</AppButton>
          <AppButton variant="secondary" size="sm" icon-only @click="dismiss" :aria-label="t('updates.dismiss')">
            <template #icon><AppIcon :icon="X" :size="12" /></template>
          </AppButton>
        </template>
        <template v-else-if="kind === 'downloading' || kind === 'verifying'">
          <AppButton variant="secondary" size="sm" @click="cancel">{{ t('common.cancel') }}</AppButton>
        </template>
        <template v-else-if="kind === 'installing'">
          <AppButton variant="primary" size="sm" disabled>{{ t('updates.installing', { version: state.latest }) }}</AppButton>
        </template>
        <template v-else-if="kind === 'ready'">
          <AppButton variant="primary" size="sm" @click="install">{{ t('updates.install') }}</AppButton>
          <AppButton variant="secondary" size="sm" icon-only @click="dismiss" :aria-label="t('updates.dismiss')">
            <template #icon><AppIcon :icon="X" :size="12" /></template>
          </AppButton>
        </template>
        <template v-else-if="kind === 'error'">
          <AppButton variant="secondary" size="sm" @click="check">{{ t('common.retry') }}</AppButton>
          <AppButton variant="secondary" size="sm" icon-only @click="dismiss" :aria-label="t('updates.dismiss')">
            <template #icon><AppIcon :icon="X" :size="12" /></template>
          </AppButton>
        </template>
        <template v-else>
          <AppButton variant="secondary" size="sm" icon-only @click="dismiss" :aria-label="t('updates.dismiss')">
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
