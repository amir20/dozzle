<template>
  <DiscoveryHint
    v-if="show"
    :title="$t('link-hint.title')"
    docs="https://dozzle.dev/guide/container-links"
    @dismiss="dismiss"
  >
    <template #icon><mdi:link-variant-plus class="size-4" /></template>

    <p class="text-base-content/80 mb-2 leading-relaxed">{{ $t("link-hint.description") }}</p>

    <div v-if="ports.length > 1" class="mb-2 flex flex-wrap gap-1">
      <button
        v-for="port in ports"
        :key="port"
        type="button"
        class="btn btn-xs"
        :class="port === selectedPort ? 'btn-primary' : 'btn-ghost border-base-content/20 border'"
        @click="selectedPort = port"
      >
        {{ port }}
      </button>
    </div>

    <pre
      class="bg-base-300/60 rounded-box overflow-x-auto p-2 font-mono text-[11px] leading-relaxed"
    ><code>{{ snippet }}</code></pre>

    <template #action>
      <button type="button" class="btn btn-xs btn-primary" @click="copySnippet">
        <mdi:content-copy class="size-3" />
        {{ $t("link-hint.copy") }}
      </button>
    </template>
  </DiscoveryHint>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";

const { container } = defineProps<{ container: Container }>();

const { t } = useI18n();
const { copy, copied, isSupported } = useClipboard({ legacy: true });
const { showToast } = useToast();

const ports = computed(() => container.publishedPorts);
const show = computed(() => !dismissedLinkHint.value && !container.url && ports.value.length > 0);

const selectedPort = ref(ports.value[0]);
watch(ports, (value) => {
  if (!value.includes(selectedPort.value)) selectedPort.value = value[0];
});

// The browser's own hostname is the best guess we have for an address that reaches the
// container. It only ever prefills the snippet, it is never rendered as a link.
const snippet = computed(
  () => `labels:
  dev.dozzle.url: http://${window.location.hostname}:${selectedPort.value}`,
);

function dismiss() {
  dismissedLinkHint.value = true;
}

async function copySnippet() {
  if (!isSupported.value) return;
  await copy(snippet.value);
  if (copied.value) {
    showToast({ title: t("toasts.copied.title"), message: t("toasts.copied.message"), type: "info" }, { expire: 2000 });
  }
}
</script>
