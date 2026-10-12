<template>
  <li v-if="nested">
    <details>
      <summary>
        <mdi:clock-outline />
        {{ $t("time-range.title") }}
        <span class="value">{{ label }}</span>
      </summary>
      <ul class="menu">
        <LogTimeRows :container :range :anchor />
      </ul>
    </details>
  </li>
  <template v-else>
    <li v-for="row in rows" :key="row.key">
      <a @click="row.run()">
        <mdi:check class="w-4" v-if="row.active" />
        <div v-else class="w-4"></div>
        {{ row.label }}
      </a>
    </li>
    <li>
      <a @click="custom()">
        <mdi:calendar-range class="w-4" />
        {{ $t("time-range.custom") }}…
      </a>
    </li>
  </template>
</template>

<script lang="ts" setup>
import type { Container } from "@/models/Container";
import type { TimeRange } from "@/composable/logs/timeRange";

// The time range rows, shared by the filter chip and the phone's actions menu, where
// `nested` folds them under one row. Renders bare `li`s, like LogFilterRows.
const { container, range, anchor, nested } = defineProps<{
  container: Container;
  range: TimeRange;
  /** The moment a frozen view was opened on, when it is not a range. */
  anchor?: Date;
  nested?: boolean;
}>();

const { rows, label, custom } = useTimeRangeMenu(
  () => container,
  () => range,
  () => anchor,
);
</script>
