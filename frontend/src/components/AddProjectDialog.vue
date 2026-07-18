<script setup>
import { ref, watch } from 'vue'
import { FolderOpen } from 'lucide-vue-next'
import { PickDirectory } from '../../wailsjs/go/main/App'
import AppButton from './ui/AppButton.vue'
import AppIcon from './ui/AppIcon.vue'

const props = defineProps({ show: Boolean })
const emit = defineEmits(['close', 'save'])

const path = ref('')

watch(() => props.show, (show) => {
  if (show) path.value = ''
})

async function pickDir() {
  const dir = await PickDirectory()
  if (dir) path.value = dir
}

function save() {
  const value = path.value.trim()
  if (!value) return
  emit('save', value)
}
</script>

<template>
  <Transition name="dlg-fade">
    <div class="mask" v-if="show" @click.self="emit('close')">
      <Transition name="dlg-pop" appear>
        <div class="dialog">
          <h3>添加项目</h3>
          <label>项目目录<input v-model="path" placeholder="/home/attson/GolandProjects/atstarter" /></label>
          <div class="inline-actions">
            <AppButton variant="secondary" @click="pickDir">
              <template #icon><AppIcon :icon="FolderOpen" :size="14" /></template>
              选择文件夹
            </AppButton>
          </div>
          <div class="btns">
            <AppButton variant="secondary" @click="emit('close')">取消</AppButton>
            <AppButton variant="primary" :disabled="!path.trim()" @click="save">添加</AppButton>
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
  background: var(--overlay);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
}

.dialog {
  width: min(520px, calc(100vw - 36px));
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  padding: var(--space-9);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface);
  box-shadow: var(--shadow-lg);
}

h3 {
  margin: 0 0 var(--space-2);
  color: var(--text);
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  letter-spacing: -0.005em;
}

label {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
}

input {
  height: 32px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  color: var(--text);
  background: var(--bg);
  padding: 0 var(--space-5);
  font: inherit;
  outline: none;
  transition: border-color var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}

input:focus {
  border-color: var(--text-subtle);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.inline-actions,
.btns {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
}
</style>
