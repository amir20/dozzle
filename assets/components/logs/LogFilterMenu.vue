<template>
  <Popover hover sheet placement="bottom-end" panel-class="log-menu-panel">
    <template #trigger>
      <!-- Glyphs rather than words, so the chip stays short however much is filtered:
           the time label, a dot per stream shown, and a funnel once levels are narrowed.
           The tooltip spells all of it out. -->
      <button
        type="button"
        class="btn btn-ghost btn-sm h-7 min-h-7 gap-1.5 px-2 font-normal"
        :title="summary"
        :aria-label="summary"
        data-testid="log-filters"
      >
        <template v-if="timed">
          <!-- Quiet while live, since that is the default; it only stands out once
               the view is narrowed to a slice of the log. -->
          <span
            class="flex items-center gap-1.5"
            :class="following ? 'text-base-content/70' : 'text-base-content'"
            data-testid="time-range"
          >
            <span v-if="following" class="bg-primary size-1.5 shrink-0 rounded-full"></span>
            <mdi:clock-outline v-else class="size-3.5 shrink-0 opacity-60" />
            <span class="text-xs whitespace-nowrap">{{ label }}</span>
          </span>
          <span class="bg-base-content/15 h-3.5 w-px"></span>
        </template>
        <span class="flex items-center gap-0.5">
          <carbon:circle-solid class="text-red size-2" v-if="streamConfig.stderr" />
          <carbon:circle-solid class="text-blue size-2" v-if="streamConfig.stdout" />
        </span>
        <span v-if="levelsFiltered" class="relative flex">
          <mdi:filter-outline class="size-3.5 opacity-70" />
          <span class="bg-primary absolute -end-0.5 -top-0.5 size-1.5 rounded-full"></span>
        </span>
      </button>
    </template>
    <ul class="menu log-menu w-full p-0">
      <template v-if="timed">
        <li class="section">{{ $t("time-range.title") }}</li>
        <LogTimeRows :container="container!" :range="range!" :anchor />
      </template>
      <li class="section">{{ $t("toolbar.section-filters") }}</li>
      <LogFilterRows />
    </ul>
  </Popover>
</template>

<script lang="ts" setup>
import type { Container } from "@/models/Container";
import type { TimeRange } from "@/composable/logs/timeRange";

// A single container view passes its container and range and gets the time rows;
// a merged view has no range to pick and leaves both out.
const { container, range, anchor } = defineProps<{
  container?: Container;
  range?: TimeRange;
  /** The moment a frozen view was opened on, when it is not a range. */
  anchor?: Date;
}>();

const timed = computed(() => !!container && !!range);

// Only read while `timed`, so the assertions never see a missing container.
const { label, following } = useTimeRangeMenu(
  () => container!,
  () => range!,
  () => anchor,
);

const { streamConfig } = useLoggingContext();
const { streamSummary, levelSummary, levelsFiltered } = useLogFilterSummary();
const { t } = useI18n();

const summary = computed(() =>
  [
    timed.value ? label.value : undefined,
    `${t("toolbar.streams")}: ${streamSummary.value}`,
    `${t("toolbar.levels")}: ${levelSummary.value}`,
  ]
    .filter(Boolean)
    .join(" · "),
);
</script>
