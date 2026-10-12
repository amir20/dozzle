<template>
  <Popover
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

      <template v-if="enableDownload">
        <li class="section">{{ $t("toolbar.section-export") }}</li>
        <li>
          <a :href="downloadUrl" download>
            <octicon:download-24 />
            {{ isFiltered ? $t("toolbar.download-filtered") : $t("toolbar.download") }}
          </a>
        </li>
      </template>

      <template v-if="workload">
        <li class="section">{{ $t("toolbar.section-workload") }}</li>
        <li>
          <button @click="rolloutRestart(workload)" :disabled="restarting">
            <carbon:renew :class="{ 'animate-spin': restarting, 'text-secondary': restarting }" />
            {{ $t("toolbar.rollout-restart", { kind: workload.kind }) }}
          </button>
        </li>
      </template>
    </ul>
  </Popover>
</template>

<script lang="ts" setup>
import type { K8sWorkload } from "@/composable/containers/rolloutRestart";

const { showSearch } = useSearchFilter();
const { enableDownload } = config;
const clear = defineEmit();

// A k8s workload adds a rollout restart; every other merged view leaves it unset.
const { name, workload } = defineProps<{ name?: string; workload?: K8sWorkload }>();
const { restarting, rolloutRestart } = useRolloutRestart();

const { streamConfig, showHostname, showContainerName, containers, levels } = useLoggingContext();

const { downloadUrl, isFiltered } = useDownloadUrl(containers, streamConfig, levels, name);
</script>
