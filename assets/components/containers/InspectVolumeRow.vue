<template>
  <li>
    <button
      type="button"
      class="hover:bg-base-300/50 flex w-full items-center gap-3 px-4 py-3 text-left transition-colors"
      :aria-expanded="open"
      @click="toggle"
    >
      <mdi:chevron-down class="size-4 shrink-0 opacity-60 transition-transform" :class="{ '-rotate-90': !open }" />
      <span class="min-w-0 flex-1 font-mono text-xs wrap-anywhere" :title="mount.source">
        <span class="text-base-content/60">{{ mount.name }}</span>
        <span class="text-base-content/40"> → </span>
        <span class="font-semibold">{{ mount.destination }}</span>
      </span>
      <span v-if="links > 1" class="status-pill status-pill-neutral shrink-0">
        {{ $t("inspect.container-count", links) }}
      </span>
      <span class="status-pill shrink-0" :class="mount.rw ? 'status-pill-neutral' : 'status-pill-secondary'">
        {{ mount.rw ? "rw" : "ro" }}
      </span>
    </button>

    <div v-if="open" class="space-y-4 px-4 pb-4 pl-11">
      <p v-if="error" class="text-base-content/60 text-xs">{{ $t("inspect.details-failed") }}</p>
      <template v-else-if="details">
        <dl class="grid grid-cols-2 gap-x-6 gap-y-1">
          <div class="min-w-0">
            <dt class="text-base-content/60 text-xs">{{ $t("inspect.driver") }}</dt>
            <dd class="mt-1 truncate font-mono text-sm">{{ details.driver }}</dd>
          </div>
          <div class="min-w-0">
            <dt class="text-base-content/60 text-xs">{{ $t("inspect.scope") }}</dt>
            <dd class="mt-1 truncate font-mono text-sm">{{ details.scope }}</dd>
          </div>
        </dl>
        <!-- Driver options are where an NFS or CIFS volume says where its data really lives. -->
        <div v-if="options.length" class="space-y-1">
          <h4 class="text-base-content/60 text-xs">{{ $t("inspect.driver-options") }}</h4>
          <ul class="font-mono text-xs">
            <li v-for="[key, value] in options" :key="key" class="wrap-anywhere">
              <span class="text-base-content/60">{{ key }}</span>
              <span class="text-base-content/40">=</span>
              <span>{{ value }}</span>
            </li>
          </ul>
        </div>
        <div class="space-y-1">
          <h4 class="text-base-content/60 text-xs">{{ $t("inspect.mounted-by") }}</h4>
          <InspectContainerList :containers="details.containers" :current-id="container.id" class="-mx-2 text-sm" />
        </div>
        <details v-if="labels.length" class="group">
          <summary
            class="text-base-content/60 hover:text-base-content flex items-center gap-1 text-xs transition-colors"
          >
            <mdi:chevron-right class="size-3.5 transition-transform group-open:rotate-90" />
            {{ $t("inspect.labels") }} ({{ labels.length }})
          </summary>
          <ul class="mt-1 font-mono text-xs">
            <li v-for="[key, value] in labels" :key="key" class="wrap-anywhere">
              <span class="text-base-content/60">{{ key }}</span>
              <span class="text-base-content/40">=</span>
              <span>{{ value }}</span>
            </li>
          </ul>
        </details>
      </template>
      <div v-else class="flex justify-center py-2">
        <span class="loading loading-spinner loading-sm opacity-60"></span>
      </div>
    </div>
  </li>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import type { ContainerMount } from "@/types/Container";

interface VolumeDetails {
  driver: string;
  scope: string;
  options?: Record<string, string>;
  labels?: Record<string, string>;
  containers: { id: string; name: string }[];
}

const { container, mount } = defineProps<{ container: Container; mount: ContainerMount & { name: string } }>();

const open = ref(false);
const details = ref<VolumeDetails>();
const error = ref(false);

// Loaded on first expand: a container can mount many volumes, and each read lists
// the host's containers to find who else mounts it.
async function load() {
  try {
    const response = await fetch(
      withBase(`/api/hosts/${container.host}/containers/${container.id}/volumes/${encodeURIComponent(mount.name)}`),
    );
    if (!response.ok) throw new Error(await response.text());
    details.value = await response.json();
  } catch (e) {
    console.error(e);
    error.value = true;
  }
}

function toggle() {
  open.value = !open.value;
  if (open.value && !details.value) {
    error.value = false;
    load();
  }
}

// Before the volume is opened, the size monitor's count is all there is; after, the
// read is current and wins.
const links = computed(
  () => details.value?.containers.length ?? container.volumes.find((v) => v.name === mount.name)?.links ?? 0,
);

const sortEntries = (record?: Record<string, string>) =>
  Object.entries(record ?? {}).sort(([a], [b]) => a.localeCompare(b));
const options = computed(() => sortEntries(details.value?.options));
const labels = computed(() => sortEntries(details.value?.labels));
</script>
