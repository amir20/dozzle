<template>
  <div class="flex flex-1 items-center gap-3 p-2" :data-edge="logEntry.edge">
    <div class="bg-base-content/10 h-px flex-1"></div>
    <div class="text-base-content/60 flex flex-wrap items-center justify-center gap-1.5 text-xs">
      <span>
        {{ logEntry.edge === "start" ? $t("time-range.edge-start") : $t("time-range.edge-end") }}
        <time class="text-base-content/80 font-mono" :datetime="logEntry.date.toISOString()">{{ time }}</time>
      </span>
      <span class="text-base-content/40" v-if="logEntry.actions.length">·</span>
      <span v-if="logEntry.actions.length">
        {{ logEntry.edge === "start" ? $t("time-range.show-earlier") : $t("time-range.show-later") }}
      </span>
      <button
        v-for="action in logEntry.actions"
        :key="action.label"
        type="button"
        class="btn btn-xs border-base-content/15 font-mono font-normal"
        @click="action.run()"
      >
        {{ action.label }}
      </button>
    </div>
    <div class="bg-base-content/10 h-px flex-1"></div>
  </div>
</template>
<script lang="ts" setup>
import { RangeEdgeLogEntry } from "@/models/LogEntry";

const { logEntry } = defineProps<{
  logEntry: RangeEdgeLogEntry;
}>();

const time = computed(() => logEntry.date.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "medium" }));
</script>
