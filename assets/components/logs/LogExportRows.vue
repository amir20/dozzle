<template>
  <li class="section">{{ $t("toolbar.section-export") }}</li>
  <li v-if="enableDownload">
    <a @click="downloadLogs()">
      <octicon:download-24 />
      {{ isFiltered ? $t("toolbar.download-filtered") : $t("toolbar.download") }}
      <KeyShortcut v-if="shortcuts" char="d" :modifiers="[]" />
    </a>
  </li>
  <li>
    <a @click="copyLogs()">
      <mdi:content-copy />
      {{ isFiltered ? $t("toolbar.copy-filtered-logs") : $t("toolbar.copy-logs") }}
      <KeyShortcut v-if="shortcuts" char="c" :modifiers="[]" />
    </a>
  </li>
</template>

<script lang="ts" setup>
import type { Container } from "@/models/Container";

// The Export section's copy and download rows, shared by both log actions menus.
// Renders bare `li`s inside a `ul.menu.log-menu`. `shortcuts` is off on a pinned
// column, so one key press does not copy every column.
const {
  containers,
  name,
  shortcuts = true,
} = defineProps<{
  containers: Container[];
  name?: string;
  shortcuts?: boolean;
}>();

const { enableDownload } = config;
const { isFiltered, copyLogs, downloadLogs } = useContainerLogs(
  () => containers,
  () => name,
);

useLetterShortcut("c", () => copyLogs(), shortcuts);
useLetterShortcut("d", downloadLogs, shortcuts && enableDownload);
</script>
