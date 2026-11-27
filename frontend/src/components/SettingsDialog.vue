<script setup>
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppButton from './ui/AppButton.vue'
import { normalizeAppSettings, terminalFontFamily } from '../appSettings.js'

const props = defineProps({
  show: Boolean,
  settings: Object,
  saving: Boolean,
  error: String,
})
const emit = defineEmits(['save', 'close'])
const { t } = useI18n()

const fontOptions = ['monospace']
const languageChoice = ref('system')
const shellChoice = ref('')
const customShell = ref('')
const fontChoice = ref('monospace')
const customFont = ref('')
const fontSize = ref(11)
const letterSpacing = ref(0)

function reset(value) {
  const settings = normalizeAppSettings(value)
  languageChoice.value = settings.language
  const knownShell = !settings.shell || settings.availableShells.includes(settings.shell)
  shellChoice.value = knownShell ? settings.shell : '__custom__'
  customShell.value = knownShell ? '' : settings.shell
  const knownFont = fontOptions.includes(settings.logFontFamily)
  fontChoice.value = knownFont ? settings.logFontFamily : '__custom__'
  customFont.value = knownFont ? '' : settings.logFontFamily
  fontSize.value = settings.logFontSize
  letterSpacing.value = settings.logLetterSpacing
}

watch(() => [props.show, props.settings], ([show]) => {
  if (show) reset(props.settings)
}, { immediate: true, deep: true })

const chosenFont = computed(() => fontChoice.value === '__custom__' ? customFont.value.trim() : fontChoice.value)
const canSave = computed(() => {
  if (shellChoice.value === '__custom__' && !customShell.value.trim()) return false
  return !!chosenFont.value && fontSize.value >= 9 && fontSize.value <= 20 && letterSpacing.value >= 0 && letterSpacing.value <= 4
})
const previewStyle = computed(() => ({
  fontFamily: terminalFontFamily(chosenFont.value),
  fontSize: `${fontSize.value}px`,
  letterSpacing: `${letterSpacing.value}px`,
}))

function save() {
  if (!canSave.value) return
  emit('save', {
    ...normalizeAppSettings(props.settings),
    language: languageChoice.value,
    shell: shellChoice.value === '__custom__' ? customShell.value.trim() : shellChoice.value,
    logFontFamily: chosenFont.value,
    logFontSize: Number(fontSize.value),
    logLetterSpacing: Number(letterSpacing.value),
  })
}
</script>

<template>
  <Transition name="dlg-fade">
    <div v-if="show" class="mask" @click.self="emit('close')">
      <Transition name="dlg-pop" appear>
        <div class="dialog" role="dialog" aria-modal="true" aria-labelledby="settings-title">
          <div class="dialog-head">
            <h3 id="settings-title">{{ t('settings.title') }}</h3>
          </div>

          <section>
            <h4>{{ t('settings.general') }}</h4>
            <label>
              <span>{{ t('settings.language') }}</span>
              <select v-model="languageChoice">
                <option value="system">{{ t('settings.languageSystem') }}</option>
                <option value="zh-CN">{{ t('settings.languageChinese') }}</option>
                <option value="en">{{ t('settings.languageEnglish') }}</option>
              </select>
            </label>
          </section>

          <section v-if="settings?.autoShell || (settings?.availableShells || []).length">
            <h4>{{ t('settings.runtime') }}</h4>
            <label>
              <span>{{ t('settings.defaultShell') }}</span>
              <select v-model="shellChoice">
                <option value="">{{ t('settings.shellAutomatic') }}{{ settings?.autoShell ? ` (${settings.autoShell})` : '' }}</option>
                <option v-for="shell in settings?.availableShells || []" :key="shell" :value="shell">{{ shell }}</option>
                <option value="__custom__">{{ t('settings.customShell') }}</option>
              </select>
            </label>
            <input v-if="shellChoice === '__custom__'" v-model="customShell" class="mono" :placeholder="t('settings.shellPathHint')" />
          </section>

          <section>
            <h4>{{ t('settings.logDisplay') }}</h4>
            <div class="field-grid">
              <label class="font-field">
                <span>{{ t('settings.font') }}</span>
                <select v-model="fontChoice">
                  <option v-for="font in fontOptions" :key="font" :value="font">{{ font === 'monospace' ? t('settings.systemMonospace') : font }}</option>
                  <option value="__custom__">{{ t('settings.customFont') }}</option>
                </select>
              </label>
              <label>
                <span>{{ t('settings.fontSize') }}</span>
                <input v-model.number="fontSize" type="number" min="9" max="20" step="1" />
              </label>
              <label>
                <span>{{ t('settings.letterSpacing') }}</span>
                <input v-model.number="letterSpacing" type="number" min="0" max="4" step="1" />
              </label>
            </div>
            <input v-if="fontChoice === '__custom__'" v-model="customFont" :placeholder="t('settings.fontFamilyHint')" />
            <div class="preview" :style="previewStyle">{{ t('settings.preview') }}</div>
          </section>

          <p v-if="error" class="error">{{ error }}</p>
          <div class="actions">
            <AppButton variant="secondary" :disabled="saving" @click="emit('close')">{{ t('common.cancel') }}</AppButton>
            <AppButton variant="primary" :disabled="saving || !canSave" @click="save">{{ saving ? t('common.saving') : t('common.save') }}</AppButton>
          </div>
        </div>
      </Transition>
    </div>
  </Transition>
</template>

<style scoped>
.mask {
  position: fixed;
  inset: 0;
  z-index: var(--z-modal);
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--overlay);
  backdrop-filter: blur(4px);
}

.dialog {
  width: min(560px, calc(100vw - 36px));
  max-height: calc(100vh - 36px);
  box-sizing: border-box;
  overflow-y: auto;
  padding: var(--space-9);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-lg);
  background: var(--surface);
  box-shadow: var(--shadow-lg);
}

.dialog-head { display: flex; align-items: center; }
h3 { margin: 0; color: var(--text); font-size: var(--fs-md); font-weight: var(--fw-semibold); }
section { padding: var(--space-8) 0; border-bottom: 1px solid var(--border); }
h4 { margin: 0 0 var(--space-6); color: var(--text-secondary); font-size: var(--fs-sm); font-weight: var(--fw-semibold); }
label { display: flex; flex-direction: column; gap: var(--space-3); min-width: 0; color: var(--text-muted); font-size: var(--fs-xs); }

select,
input {
  height: 34px;
  box-sizing: border-box;
  width: 100%;
  padding: 0 var(--space-5);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  outline: 0;
  background: var(--bg);
  color: var(--text);
  font: inherit;
  font-size: var(--fs-sm);
}
select:focus,
input:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--focus-ring); }
section > input { margin-top: var(--space-4); }
.mono { font-family: var(--font-mono); }
.field-grid { display: grid; grid-template-columns: minmax(0, 1fr) 88px 104px; gap: var(--space-5); }

.preview {
  margin-top: var(--space-6);
  min-height: 42px;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  overflow: hidden;
  padding: 0 var(--space-6);
  white-space: nowrap;
  border: 1px solid var(--log-border);
  border-radius: var(--radius-md);
  background: var(--log-bg);
  color: var(--log-text);
}

.error { margin: var(--space-5) 0 0; color: var(--danger); font-size: var(--fs-sm); }
.actions { display: flex; justify-content: flex-end; gap: var(--space-3); padding-top: var(--space-7); }

@media (max-width: 560px) {
  .field-grid { grid-template-columns: 1fr 1fr; }
  .font-field { grid-column: 1 / -1; }
}
</style>
