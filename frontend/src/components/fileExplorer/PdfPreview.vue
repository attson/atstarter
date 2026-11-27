<script lang="ts" setup>
import { useI18n } from "vue-i18n";
import { onBeforeUnmount, ref, watch } from "vue";
import BinaryBanner from "./BinaryBanner.vue";
import type { FileSystemBridge } from "./fsBridge";

const { t } = useI18n();

const props = defineProps<{ fs: FileSystemBridge; path: string }>();
const src = ref("");
const failed = ref(false);
const error = ref("");
let active = true;
let request = 0;
let requested: { fs: FileSystemBridge; path: string } | null = null;

function releaseAsset() {
  if (!requested) return;
  requested.fs.revokeAssetUrl?.(requested.path);
  requested = null;
}

async function loadAsset() {
  const fs = props.fs;
  const path = props.path;
  const currentRequest = ++request;
  releaseAsset();
  requested = { fs, path };
  src.value = "";
  failed.value = false;
  error.value = "";
  try {
    const url = await fs.assetUrlFor(path);
    if (!active || currentRequest !== request || props.fs !== fs || props.path !== path) {
      fs.revokeAssetUrl?.(path);
      return;
    }
    src.value = url;
  } catch (err) {
    if (active && currentRequest === request && props.fs === fs && props.path === path) {
      error.value = (typeof err === "string" ? err : (err as Error)?.message) || "";
      failed.value = true;
    }
  }
}

watch(() => [props.path, props.fs.identity], () => { void loadAsset(); }, { immediate: true });
onBeforeUnmount(() => {
  active = false;
  request++;
  releaseAsset();
});
</script>

<template>
  <BinaryBanner v-if="failed" :fs="fs" :path="path" :message="error || undefined" />
  <div v-else class="pdf-host">
    <object :data="src" type="application/pdf" :aria-label="t('files.pdfPreview')">
      <!-- Shown if the browser can't render PDFs natively. -->
      <BinaryBanner :fs="fs" :path="path" />
    </object>
  </div>
</template>

<style scoped>
.pdf-host {
  flex: 1 1 auto;
  display: flex;
  background: var(--ed-editor-bg, #22272e);
}
object {
  flex: 1 1 auto;
  width: 100%;
  height: 100%;
  border: none;
}
</style>
