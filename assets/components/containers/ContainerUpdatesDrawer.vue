<template>
  <div class="flex min-h-full flex-col">
    <div class="space-y-6 p-4 pb-8">
      <div class="pr-20">
        <h2 class="text-2xl font-bold">{{ $t("updates.title") }}</h2>
        <p class="text-base-content/60">
          {{ showingJob ? $t("updates.subtitle-running") : $t("updates.subtitle") }}
        </p>
      </div>

      <InlineNotice v-if="job?.running && job.trigger === 'schedule'" type="info">
        {{ $t("updates.scheduled-running") }}
      </InlineNotice>

      <!-- The job's own list once one is underway: its containers are being
           recreated under new ids, so the store no longer knows them by the
           ids this drawer was opened with. -->
      <div
        v-if="showingJob && job"
        class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border"
      >
        <div v-for="item in job.items" :key="`${item.host}/${item.id}`" class="flex items-start gap-3 p-4">
          <div class="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-full" :class="tint(item)">
            <span
              v-if="!isFinished(item.status) && item.status !== 'queued'"
              class="loading loading-spinner size-3.5"
            ></span>
            <mdi:check v-else-if="item.status === 'done' || item.status === 'up-to-date'" class="size-4" />
            <mdi:alert-circle-outline v-else-if="item.status === 'error'" class="size-4" />
            <mdi:restore v-else-if="item.status === 'rolled-back'" class="size-4" />
            <mdi:clock-outline v-else class="size-4" />
          </div>
          <div class="min-w-0 flex-1">
            <div class="truncate text-sm font-medium">{{ item.name }}</div>
            <div class="text-base-content/60 truncate font-mono text-xs">{{ item.image }}</div>
            <div v-if="item.error" class="text-base-content/60 mt-1 text-xs wrap-anywhere">{{ item.error }}</div>
            <!-- The container now runs under a new id; whatever holds its name is
                 what a rollback acts on, and only while it still has a target. -->
            <button
              v-if="item.status === 'done' && rollbackCandidate(item)"
              type="button"
              class="text-base-content/60 hover:text-base-content mt-1 inline-flex items-center gap-1 text-xs transition-colors"
              @click="requestRollback(rollbackCandidate(item)!)"
            >
              <mdi:restore class="size-3.5" />
              {{ $t("rollback.link") }}
            </button>
          </div>
          <span class="status-pill shrink-0" :class="pill(item.status)">
            {{ $t(`updates.status.${item.status}`) }}
            <template v-if="item.status === 'pulling' && item.total">
              {{ Math.round(((item.current ?? 0) / item.total) * 100) }}%
            </template>
          </span>
        </div>
      </div>

      <template v-else>
        <div
          v-if="candidates.length"
          class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border"
        >
          <label
            v-for="container in candidates"
            :key="container.id"
            :ref="(el) => container.id === focus && (focusEl = el as HTMLElement)"
            class="hover:bg-base-300/40 flex items-start gap-3 p-4 transition-colors"
          >
            <input v-model="selected" type="checkbox" class="checkbox checkbox-sm mt-0.5" :value="container.id" />
            <ContainerIcon :state="container.state" :health="container.health" :slug="container.icon" class="size-6" />
            <div class="min-w-0 flex-1">
              <div class="truncate text-sm font-medium">{{ container.name }}</div>
              <div class="text-base-content/60 truncate font-mono text-xs">
                {{ container.image }}
                <template v-if="multipleHosts">
                  <span class="text-base-content/40"> · {{ container.hostLabel }}</span>
                </template>
              </div>
            </div>
            <span
              v-if="autoUpdateEnabled(container)"
              class="status-pill status-pill-neutral shrink-0"
              :title="$t('updates.auto-hint')"
            >
              {{ $t("updates.auto") }}
            </span>
          </label>
        </div>
        <p v-else class="text-base-content/60 text-sm">{{ $t("updates.none") }}</p>
      </template>

      <p class="text-base-content/40 text-xs">
        <i18n-t keypath="updates.auto-footnote">
          <template #label>
            <code class="font-mono">{{ AUTO_UPDATE_LABEL }}=true</code>
          </template>
        </i18n-t>
      </p>
    </div>

    <div
      class="bg-base-100 border-base-content/10 sticky bottom-0 z-10 -mx-4 mt-auto flex flex-wrap items-center gap-2 border-t p-4"
    >
      <span v-if="!showingJob && selfSelected" class="text-base-content/60 mr-auto text-xs">
        {{ $t("updates.self-last") }}
      </span>
      <span v-else-if="showingJob && job" class="text-base-content/60 mr-auto font-mono text-xs">
        {{ $t("updates.progress", { done: finishedCount, total: job.items.length }) }}
      </span>
      <div class="ml-auto flex items-center gap-2">
        <button
          v-if="!showingJob && config.imageCheckMode !== 'off'"
          type="button"
          class="btn btn-sm"
          :disabled="checking"
          @click="checkAll(true)"
        >
          <span v-if="checking" class="loading loading-spinner size-3.5"></span>
          {{ $t("toolbar.check-for-updates") }}
        </button>
        <button v-if="showingJob && !running" type="button" class="btn btn-sm" @click="showingJob = false">
          {{ $t("updates.back") }}
        </button>
        <button
          v-if="!showingJob"
          type="button"
          class="btn btn-primary btn-sm"
          :disabled="selectedContainers.length === 0 || running"
          @click="updateSelected"
        >
          {{ $t("updates.update-selected", selectedContainers.length) }}
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import {
  AUTO_UPDATE_LABEL,
  type BulkUpdateItem,
  type BulkUpdateStatus,
  autoUpdateEnabled,
  isFinished,
} from "@/composable/containers/bulkUpdate";

const { focus } = defineProps<{ focus?: string }>();

const { containers } = storeToRefs(useContainerStore()) as unknown as { containers: Ref<Container[]> };
const { hosts } = useHosts();
const { checkAll, checking, hasUpdate, isSelf } = useImageUpdates();
const { job, running, start, hold } = useBulkUpdate();

// Watching for as long as the drawer is open, so a job started elsewhere (the
// schedule, another tab) shows up here too.
const release = hold();
onScopeDispose(release);

const multipleHosts = computed(() => Object.keys(hosts.value).length > 1);

const candidates = computed(() => containers.value.filter(hasUpdate).sort((a, b) => a.name.localeCompare(b.name)));

// Running containers are selected to begin with, and so is one that appears
// after the drawer opened (a check finishing). A stopped one is listed but left
// for the user to opt in, since it may be stopped on purpose.
const selected = ref<string[]>([]);
const seen = new Set<string>();
watch(
  candidates,
  (list) => {
    const fresh = list.filter((c) => !seen.has(c.id));
    fresh.forEach((c) => seen.add(c.id));
    const running = fresh.filter((c) => c.state === "running").map((c) => c.id);
    if (running.length) selected.value = [...selected.value, ...running];
  },
  { immediate: true },
);

const selectedContainers = computed(() => candidates.value.filter((c) => selected.value.includes(c.id)));
const selfSelected = computed(() => selectedContainers.value.some(isSelf));

const showingJob = ref(!!job.value?.running);
watch(running, (now) => now && (showingJob.value = true));

const finishedCount = computed(() => job.value?.items.filter((i) => isFinished(i.status)).length ?? 0);

const focusEl = ref<HTMLElement>();
onMounted(() => focusEl.value?.scrollIntoView({ block: "center" }));

// The container that holds a finished item's name now.
function successor(item: BulkUpdateItem) {
  if (!config.enableActions || item.status !== "done") return undefined;
  return containers.value.find((c) => c.host === item.host && c.name === item.name && c.state !== "deleted");
}

// The successor, if the host knows where a rollback would take it.
function rollbackCandidate(item: BulkUpdateItem) {
  const c = successor(item);
  return c && rollbackTargetOf(c) && !isRollingBack(c) ? c : undefined;
}

// Asks for each successor's target as it appears.
watchEffect(() => {
  for (const item of job.value?.items ?? []) {
    const c = successor(item);
    if (c) loadRollbackTarget(c);
  }
});

function updateSelected() {
  showingJob.value = true;
  start(selectedContainers.value);
}

function pill(status: BulkUpdateStatus) {
  switch (status) {
    case "done":
      return "status-pill-success";
    case "error":
      return "status-pill-error";
    case "rolled-back":
      return "status-pill-warning";
    case "pulling":
    case "recreating":
    case "verifying":
      return "status-pill-primary";
    default:
      return "status-pill-neutral";
  }
}

function tint(item: BulkUpdateItem) {
  switch (item.status) {
    case "done":
      return "bg-success/10 text-success";
    case "error":
      return "bg-error/10 text-error";
    case "rolled-back":
      return "bg-warning/10 text-warning";
    case "queued":
    case "up-to-date":
      return "bg-base-content/5 text-base-content/60";
    default:
      return "bg-info/10 text-info";
  }
}
</script>
