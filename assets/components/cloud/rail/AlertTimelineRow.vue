<template>
  <div :data-row="row.key" class="transition-colors" :class="{ 'bg-base-300': highlighted }">
    <button
      type="button"
      class="hover:bg-base-300 flex w-full items-start gap-3 p-3 text-left transition-colors"
      :aria-expanded="open"
      @click="open = !open"
    >
      <!-- Severity rides the icon; the row behind it stays neutral. The quiet
           row has no severity to carry, so its glyph is muted. -->
      <span
        class="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-full"
        :class="row.kind === 'problem' ? levelTint(row.level) : 'bg-base-content/10 text-base-content/60'"
      >
        <mdi:alert-circle-outline v-if="row.kind === 'problem'" class="size-4" />
        <mdi:text-search v-else class="size-4" />
      </span>

      <span class="min-w-0 flex-1">
        <span v-if="row.kind === 'problem'" class="flex flex-wrap items-baseline gap-x-2 gap-y-1">
          <span class="text-sm font-semibold wrap-anywhere">{{ row.headline }}</span>
          <span class="status-pill" :class="levelPill(row.level)">{{ row.level }}</span>
        </span>
        <span v-else class="text-base-content/60 block text-sm" data-quiet>
          {{
            $t(
              "cloud-rail.alerts-quiet",
              { n: row.happened, containers: $t("cloud-rail.alerts-quiet-containers", row.containers.length) },
              row.happened,
            )
          }}
        </span>

        <span class="text-base-content/60 mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
          <span v-if="!hideContainer" class="max-w-full truncate font-mono">{{ containerList }}</span>
          <RelativeTime :date="new Date(row.latest / 1e6)" />
          <template v-if="row.kind === 'problem'">
            <span data-happened>{{ $t("cloud-rail.alerts-happened", row.happened) }}</span>
            <span data-notified>
              {{
                row.notified ? $t("cloud-rail.alerts-notified", row.notified) : $t("cloud-rail.alerts-notified-earlier")
              }}
            </span>
          </template>
        </span>
      </span>

      <mdi:chevron-down
        class="mt-1.5 size-4 shrink-0 opacity-40 transition-transform"
        :class="{ 'rotate-180': open }"
      />
    </button>

    <!-- Newest first, each with the jump to its own moment. On a phone the line
         gets a row of its own; beside the time and name it had a few letters. -->
    <ul v-if="open" class="divide-base-content/10 border-base-content/10 divide-y border-t" data-occurrences>
      <li
        v-for="(o, i) in row.occurrences"
        :key="i"
        class="flex flex-wrap items-center gap-x-3 py-1 pr-2 pl-13 text-xs"
      >
        <span class="text-base-content/60 shrink-0 font-mono tabular-nums">{{
          clock.format(new Date(o.ts / 1e6))
        }}</span>
        <span v-if="!hideContainer" class="text-base-content/60 max-w-32 shrink-0 truncate font-mono">
          {{ nameOf(o.containerId) }}
        </span>
        <span
          class="min-w-0 flex-1 truncate font-mono max-sm:order-last max-sm:basis-full max-sm:pb-1"
          :title="o.text"
          >{{ o.text }}</span
        >
        <AlertRowActions
          class="max-sm:ml-auto"
          v-if="readable(o.containerId)"
          :moment="{ containerId: o.containerId, date: new Date(o.ts / 1e6), logId: o.logId }"
        />
      </li>
      <li v-if="row.kind === 'problem' && row.url" class="p-1.5 pl-11">
        <a
          :href="row.url"
          target="_blank"
          rel="noreferrer noopener"
          class="hover:bg-base-300 flex items-center gap-2 rounded-md px-2 py-1.5 text-xs transition-colors"
        >
          <mdi:cloud-outline class="size-4 shrink-0 opacity-60" />
          <span class="min-w-0 flex-1 truncate">{{ $t("notifications.history.open-in-cloud") }}</span>
          <mdi:open-in-new class="size-3.5 shrink-0 opacity-40" />
        </a>
      </li>
    </ul>
  </div>
</template>

<script lang="ts" setup>
import type { TimelineRow } from "@/composable/cloud/alertsTimeline";
import { levelPill, levelTint } from "@/composable/cloud/alertsTimeline";

const {
  row,
  hideContainer = false,
  highlighted = false,
} = defineProps<{ row: TimelineRow; hideContainer?: boolean; highlighted?: boolean }>();

const open = ref(false);
const store = useContainerStore();
const clock = useMetricsClock({ hour: "2-digit", minute: "2-digit" });

// A container this instance no longer sees keeps a short id rather than a blank.
const nameOf = (id: string) => store.allContainersById[id]?.name ?? id.slice(0, 12);
// Jumping to a container that is gone lands on Not Found.
const readable = (id: string) => !!store.allContainersById[id];

// Two names, then a count: a problem across a whole stack would otherwise push
// the rest of the line off the row.
const containerList = computed(() => {
  const names = row.containers.map(nameOf);
  return names.length > 2 ? `${names.slice(0, 2).join(", ")} +${names.length - 2}` : names.join(", ");
});
</script>
