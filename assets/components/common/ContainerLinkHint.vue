<template>
  <DiscoveryHint
    v-if="show"
    :title="$t('link-hint.title')"
    docs="https://dozzle.dev/guide/container-links"
    @dismiss="dismiss"
  >
    <template #icon><mdi:link-variant-plus class="size-4" /></template>

    <p class="text-base-content/80 mb-2 leading-relaxed">{{ $t("link-hint.description") }}</p>

    <div v-if="suggestions.length > 1" class="mb-2 flex flex-wrap gap-1">
      <button
        v-for="suggestion in suggestions"
        :key="suggestion.url"
        type="button"
        class="btn btn-xs max-w-full truncate"
        :class="suggestion.url === selected ? 'btn-primary' : 'btn-ghost border-base-content/20 border'"
        @click="selected = suggestion.url"
      >
        {{ suggestion.label }}
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

// Traefik labels name an address that actually reaches the container from outside, so they
// come first. A published host port is only a guess: the browser's own hostname is the best
// stand-in we have for the docker host. Either way this only ever prefills the snippet,
// it is never rendered as a link.
const suggestions = computed(() => [
  ...container.traefikUrls.map((url) => ({ label: url.replace(/^https?:\/\//, ""), url })),
  ...container.publishedPorts.map((port) => ({
    label: `:${port}`,
    url: `http://${window.location.hostname}:${port}`,
  })),
]);

const show = computed(() => !dismissedLinkHint.value && !container.url && suggestions.value.length > 0);

const selected = ref(suggestions.value[0]?.url);
watch(suggestions, (value) => {
  if (!value.some(({ url }) => url === selected.value)) selected.value = value[0]?.url;
});

const snippet = computed(
  () => `labels:
  dev.dozzle.url: ${selected.value}`,
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
