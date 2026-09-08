<template>
  <div class="flex w-64 flex-col gap-3">
    <div class="flex items-start gap-2">
      <ContainerIcon :state="container.state" :health="container.health" :slug="container.icon" class="mt-0.5 size-5" />
      <div class="min-w-0 flex-1">
        <div class="truncate font-semibold" :title="container.name">{{ container.name }}</div>
        <div class="text-base-content/50 truncate font-mono text-[11px]" :title="imageTag">{{ imageTag }}</div>
      </div>
      <ContainerHealth v-if="container.health" :health="container.health" />
    </div>

    <div v-if="isRunning" class="grid grid-cols-[auto_1fr] items-center gap-x-2 gap-y-1">
      <span class="text-base-content/50 text-[11px] uppercase">{{ $t("label.cpu") }}</span>
      <ContainerStatCell :container="container" type="cpu" :host="host" />
      <span class="text-base-content/50 text-[11px] uppercase">{{ $t("label.mem") }}</span>
      <ContainerStatCell :container="container" type="mem" :host="host" />
    </div>

    <div class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
      <template v-for="row in rows" :key="row.label">
        <span class="text-base-content/50 uppercase">{{ row.label }}</span>
        <span class="truncate text-right font-medium" :title="row.title ?? row.value">{{ row.value }}</span>
      </template>
      <template v-if="isRunning">
        <span class="text-base-content/50 uppercase">{{ $t("popup.started") }}</span>
        <RelativeTime :date="container.startedAt" class="truncate text-right font-medium" />
      </template>
      <template v-else-if="container.finishedAt.getFullYear() > 0">
        <span class="text-base-content/50 uppercase">{{ $t("popup.finished") }}</span>
        <RelativeTime :date="container.finishedAt" class="truncate text-right font-medium" />
      </template>
    </div>

    <!-- Everything below the hairline is a control rather than a fact about the
         container, so the URL link and the icon row share one divider. -->
    <div class="border-base-content/10 flex flex-col gap-2 border-t pt-2">
      <ContainerLink v-if="container.url" :container="container" class="text-primary! text-xs hover:underline">
        <span class="flex-1 truncate">{{ shortUrl }}</span>
      </ContainerLink>

      <!-- Icon-only, unlabeled: the popup is already dense, and every one of these
           has a labeled twin in the container toolbar for anyone hunting by name. -->
      <div class="flex items-center gap-0.5">
        <button
          type="button"
          class="nav-btn icon-btn"
          :class="{ 'text-secondary!': isPinnedColumn }"
          :title="isPinnedColumn ? $t('tooltip.unpin-column') : $t('tooltip.pin-column')"
          :aria-label="isPinnedColumn ? $t('tooltip.unpin-column') : $t('tooltip.pin-column')"
          :aria-pressed="isPinnedColumn"
          @click="togglePinnedColumn()"
        >
          <cil:columns class="size-4" />
        </button>

        <button
          type="button"
          class="nav-btn icon-btn"
          :class="{ 'text-secondary!': pinned }"
          :title="pinned ? $t('toolbar.unpin') : $t('toolbar.pin')"
          :aria-label="pinned ? $t('toolbar.unpin') : $t('toolbar.pin')"
          :aria-pressed="pinned"
          @click="pinned = !pinned"
        >
          <ph:map-pin-simple-fill v-if="pinned" class="size-4" />
          <ph:map-pin-simple v-else class="size-4" />
        </button>

        <button
          type="button"
          class="nav-btn icon-btn"
          v-if="enableShell && isRunning"
          :title="$t('toolbar.shell')"
          :aria-label="$t('toolbar.shell')"
          @click="showDrawer(Terminal, { container, action: 'exec' }, 'lg')"
        >
          <material-symbols:terminal class="size-4" />
        </button>

        <!-- aria-disabled rather than disabled: the reason it cannot be used right
             now lives in the tooltip, and a real disabled button never shows one. -->
        <button
          type="button"
          class="nav-btn icon-btn aria-disabled:cursor-default aria-disabled:opacity-40"
          :class="{ 'text-secondary!': isMerged }"
          :aria-disabled="!canMerge"
          :title="mergeTitle"
          :aria-label="mergeTitle"
          :aria-pressed="isMerged"
          @click="canMerge && toggleMerge(container.id)"
        >
          <ph:arrows-merge class="size-4" />
        </button>

        <!-- Start/stop/restart sit apart from the view controls above: one group
             changes what you are looking at, the other changes the container. -->
        <template v-if="enableActions">
          <button
            type="button"
            class="nav-btn icon-btn hover:text-error! ms-auto disabled:pointer-events-none disabled:opacity-40"
            v-if="isRunning"
            :disabled="actionStates.stop || actionStates.restart"
            :title="$t('toolbar.stop')"
            :aria-label="$t('toolbar.stop')"
            @click="stop()"
          >
            <carbon:stop-filled-alt class="size-4" />
          </button>
          <button
            type="button"
            class="nav-btn icon-btn hover:text-success! ms-auto disabled:pointer-events-none disabled:opacity-40"
            v-else
            :disabled="actionStates.start || actionStates.restart"
            :title="$t('toolbar.start')"
            :aria-label="$t('toolbar.start')"
            @click="start()"
          >
            <carbon:play class="size-4" />
          </button>

          <button
            type="button"
            class="nav-btn icon-btn disabled:pointer-events-none disabled:opacity-40"
            :disabled="actionStates.stop || actionStates.start || actionStates.restart"
            :title="$t('toolbar.restart')"
            :aria-label="$t('toolbar.restart')"
            @click="restart()"
          >
            <carbon:restart class="size-4" :class="{ 'text-secondary animate-spin': actionStates.restart }" />
          </button>
        </template>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import Terminal from "@/components/Terminal.vue";
import { useStreamedContainers } from "@/composable/streamedContainers";

const { container } = defineProps<{
  container: Container;
}>();

const { t } = useI18n();
const { hosts } = useHosts();
const { enableActions, enableShell } = config;
const showDrawer = useDrawer();
const pinnedStore = usePinnedLogsStore();
const { actionStates, start, stop, restart } = useContainerActions(toRef(() => container));

const host = computed(() => hosts.value[container.host]);
const isRunning = computed(() => container.state === "running");
const imageTag = computed(() => container.image.replace(/@sha.*/, ""));
const shortUrl = computed(() => container.url?.replace(/^https?:\/\//, "").replace(/\/$/, ""));

const { ids: streamedIds, toggle: toggleMerge } = useStreamedContainers();

const isMerged = computed(() => streamedIds.value.includes(container.id));

// Nothing to merge into from a dashboard or a group view, and removing the last
// id would leave a merged view with no containers in it.
const canMerge = computed(() => streamedIds.value.length > (isMerged.value ? 1 : 0));

const mergeTitle = computed(() => {
  if (streamedIds.value.length === 0) return t("tooltip.merge-stream-hint");
  return isMerged.value ? t("tooltip.unmerge-stream") : t("tooltip.merge-stream");
});

const isPinnedColumn = computed(() => pinnedStore.isPinned(container));
const togglePinnedColumn = () =>
  isPinnedColumn.value ? pinnedStore.unPinContainer(container) : pinnedStore.pinContainer(container);

// Name-based, like the sidebar's "Pinned" group and the title bar's pin.
const pinned = computed({
  get: () => pinnedContainers.value.has(container.name),
  set: (value) => {
    if (value) {
      pinnedContainers.value.add(container.name);
    } else {
      pinnedContainers.value.delete(container.name);
    }
  },
});

const rows = computed(() => {
  const rows: { label: string; value: string; title?: string }[] = [
    { label: t("label.status"), value: container.health ?? container.state },
  ];

  if (config.hosts.length > 1) {
    rows.push({ label: t("label.host"), value: container.hostLabel });
  }

  if (container.customGroup) {
    rows.push({ label: t("popup.group"), value: container.customGroup });
  }

  const ports = container.portMappings;
  if (ports.length > 0) {
    const mapped = ports.map(({ host, container }) => `${host}→${container}`);
    rows.push({
      label: t("popup.ports"),
      value: mapped.length > 2 ? `${mapped.slice(0, 2).join(", ")} +${mapped.length - 2}` : mapped.join(", "),
      title: mapped.join(", "),
    });
  }

  return rows;
});
</script>
