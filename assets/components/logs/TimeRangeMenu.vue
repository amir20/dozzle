<template>
  <Popover
    hover
    sheet
    placement="bottom-end"
    panel-class="rounded-box bg-base-200 border-base-content/10 w-max min-w-52 border p-1.5 shadow-lg"
  >
    <template #trigger>
      <!-- Quiet while live, since that is the default; it only stands out once the
           view is narrowed to a slice of the log. -->
      <button
        type="button"
        class="btn btn-ghost btn-sm h-7 min-h-7 gap-1.5 px-2 font-normal"
        :class="following ? 'text-base-content/70' : 'text-base-content'"
        :title="$t('time-range.title')"
        data-testid="time-range"
      >
        <span v-if="following" class="bg-primary size-1.5 shrink-0 rounded-full"></span>
        <mdi:clock-outline v-else class="size-3.5 shrink-0 opacity-60" />
        <span class="text-xs whitespace-nowrap">{{ label }}</span>
      </button>
    </template>
    <ul class="menu w-full p-0">
      <li class="section">{{ $t("time-range.title") }}</li>
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
    </ul>
  </Popover>
</template>

<script lang="ts" setup>
import type { Container } from "@/models/Container";
import type { TimeRange } from "@/composable/logs/timeRange";

const { container, range, anchor } = defineProps<{
  container: Container;
  range: TimeRange;
  /** The moment a frozen view was opened on, when it is not a range. */
  anchor?: Date;
}>();

const { rows, label, following, custom } = useTimeRangeMenu(
  () => container,
  () => range,
  () => anchor,
);
</script>
