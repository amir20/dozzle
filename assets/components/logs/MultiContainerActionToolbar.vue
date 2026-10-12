<template>
  <Popover
    v-bind="$attrs"
    hover
    sheet
    placement="bottom-end"
    panel-class="rounded-box bg-base-200 border-base-content/10 w-max min-w-60 border p-1.5 shadow-lg"
  >
    <template #trigger>
      <button type="button" class="icon-btn btn btn-ghost btn-sm w-8 px-0" data-testid="log-actions">
        <mdi:dots-vertical class="size-4" />
      </button>
    </template>
    <ul class="menu log-menu w-full p-0">
      <li class="section">{{ $t("toolbar.section-logs") }}</li>
      <li>
        <a @click="showSearch = true">
          <mdi:magnify /> {{ $t("toolbar.search") }}
          <KeyShortcut char="f" />
        </a>
      </li>
      <li>
        <a @click="clear()">
          <octicon:trash-24 /> {{ $t("toolbar.clear") }}
          <KeyShortcut char="l" :modifiers="['shift', 'meta']" />
        </a>
      </li>
      <li v-if="canAnalyze">
        <a @click="openAnalytics()">
          <ph:file-sql /> {{ $t("analytics.title") }}
          <KeyShortcut char="s" :modifiers="[]" />
        </a>
      </li>

      <!-- The filter chip beside this menu carries these, except on a phone. -->
      <template v-if="isMobile">
        <li class="section">{{ $t("toolbar.section-filters") }}</li>
        <LogFilterRows />
      </template>

      <!-- Only a merged stream mixes containers and hosts in one column, so these
           two toggles have nowhere else to live. -->
      <li class="section">{{ $t("toolbar.section-display") }}</li>
      <li>
        <a @click="showContainerName = !showContainerName">
          <mdi:check class="w-4" v-if="showContainerName" />
          <div v-else class="w-4"></div>
          {{ $t("toolbar.show-container-name") }}
        </a>
      </li>
      <li>
        <a @click="showHostname = !showHostname">
          <mdi:check class="w-4" v-if="showHostname" />
          <div v-else class="w-4"></div>
          {{ $t("toolbar.show-hostname") }}
        </a>
      </li>

      <li class="section">{{ $t("toolbar.section-export") }}</li>
      <li v-if="enableDownload">
        <a :href="downloadUrl" download>
          <octicon:download-24 />
          {{ isFiltered ? $t("toolbar.download-filtered") : $t("toolbar.download") }}
          <KeyShortcut char="d" :modifiers="[]" />
        </a>
      </li>
      <li>
        <a @click="copyLogs()">
          <mdi:content-copy />
          {{ isFiltered ? $t("toolbar.copy-filtered-logs") : $t("toolbar.copy-logs") }}
          <KeyShortcut char="c" :modifiers="[]" />
        </a>
      </li>

      <template v-if="workload || bulkRows.length">
        <li class="section">{{ workload ? $t("toolbar.section-workload") : $t("toolbar.section-containers") }}</li>
        <li v-for="plan in bulkRows" :key="plan.action">
          <button @click="pending = plan" :disabled="!!running">
            <span v-if="running === plan.action" class="loading loading-spinner loading-xs"></span>
            <component v-else :is="icons[plan.action]" />
            {{ $t(`toolbar.${plan.action}-all`) }}
          </button>
        </li>
        <li v-if="workload">
          <button @click="rolloutRestart(workload)" :disabled="restarting">
            <carbon:renew :class="{ 'animate-spin': restarting, 'text-secondary': restarting }" />
            {{ $t("toolbar.rollout-restart", { kind: workload.kind }) }}
          </button>
        </li>
      </template>
    </ul>
  </Popover>

  <!-- Outside the hover menu, which closes on pointer leave and would take the dialog with it. -->
  <dialog ref="confirm" class="modal max-md:modal-bottom" @close="pending = undefined">
    <div v-if="pending" class="modal-box flex w-full max-w-md flex-col p-0">
      <div class="flex flex-col gap-1 px-6 pt-6">
        <h2 class="text-2xl font-bold">
          {{ $t(`toolbar.bulk-confirm-${pending.action}`, pending.targets.length) }}
        </h2>
        <p v-if="pending.skippedSelf" class="text-base-content/60 text-sm">{{ $t("toolbar.bulk-self-skipped") }}</p>
      </div>
      <ul
        class="border-base-content/15 bg-base-200/40 divide-base-content/10 mx-6 my-4 divide-y rounded-lg border font-mono text-sm"
      >
        <li v-for="c in pending.targets.slice(0, MAX_NAMES)" :key="c.id" class="truncate px-3 py-1.5">{{ c.name }}</li>
        <li v-if="pending.targets.length > MAX_NAMES" class="text-base-content/40 px-3 py-1.5">
          {{ $t("toolbar.bulk-more", { count: pending.targets.length - MAX_NAMES }) }}
        </li>
      </ul>
      <div class="border-base-content/10 flex justify-end gap-2 border-t px-6 py-4">
        <button type="button" class="btn btn-sm" @click="confirm?.close()">{{ $t("button.cancel") }}</button>
        <button
          type="button"
          class="btn btn-sm"
          :class="pending.action === 'start' ? 'btn-primary' : 'text-error'"
          @click="runPending"
        >
          {{ $t(`toolbar.${pending.action}`) }}
        </button>
      </div>
    </div>
    <form method="dialog" class="modal-backdrop">
      <button>{{ $t("button.cancel") }}</button>
    </form>
  </dialog>
</template>

<script lang="ts" setup>
import type { K8sWorkload } from "@/composable/containers/rolloutRestart";
import type { BulkAction, BulkPlan } from "@/composable/containers/bulkContainerActions";
import type { Component } from "vue";
import CarbonRestart from "~icons/carbon/restart";
import CarbonStopFilledAlt from "~icons/carbon/stop-filled-alt";
import CarbonPlay from "~icons/carbon/play";
// Drawer-only, and it drags in apache-arrow; loading it eagerly put both on every log view.
const LogAnalytics = defineAsyncComponent(() => import("@/components/logs/LogAnalytics.vue"));

defineOptions({ inheritAttrs: false });

const { showSearch } = useSearchFilter();
const { enableDownload } = config;
const clear = defineEmit();

// A k8s workload adds a rollout restart; every other merged view leaves it unset.
const { name, workload } = defineProps<{ name?: string; workload?: K8sWorkload }>();
const { restarting, rolloutRestart } = useRolloutRestart();

const { showHostname, showContainerName, containers, hasComplexLogs } = useLoggingContext();

const { downloadUrl, isFiltered, copyLogs, downloadLogs } = useContainerLogs(containers, () => name);

const showDrawer = useDrawer();
const openAnalytics = () => showDrawer(LogAnalytics, { containers: toValue(containers), name }, "lg");
// hasComplexLogs outlives the containers that set it, and the drawer needs at least one.
const canAnalyze = computed(() => hasComplexLogs.value && toValue(containers).length > 0);

useLetterShortcut("d", downloadLogs, enableDownload);
useLetterShortcut("c", () => copyLogs());
useLetterShortcut("s", openAnalytics, canAnalyze);

// Restart, stop and start for every container in the view, behind a confirm.
const MAX_NAMES = 8;
const icons: Record<BulkAction, Component> = { restart: CarbonRestart, stop: CarbonStopFilledAlt, start: CarbonPlay };
const { available, running, run } = useBulkContainerActions(containers);
const bulkRows = computed(() => (config.enableActions ? available.value : []));

const confirm = useTemplateRef<HTMLDialogElement>("confirm");
const pending = shallowRef<BulkPlan>();
watch(pending, (plan) => plan && nextTick(() => confirm.value?.showModal()));

function runPending() {
  const plan = pending.value;
  confirm.value?.close();
  // Re-plan against current states: one may have changed while the dialog was open.
  if (plan) run(bulkPlan(toValue(containers), plan.action));
}
</script>
