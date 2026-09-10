<template>
  <!--
    The line the question is about, quoted where the question is being asked.

    A chip saying "this log line" would be a claim the reader has to take on
    faith; the line itself is the only thing that proves the assistant got the
    row they clicked and not its neighbour.
  -->
  <div class="flex items-start gap-2">
    <mdi:format-quote-close class="text-base-content/30 mt-px size-3.5 shrink-0" />
    <div class="min-w-0 flex-1">
      <div class="text-base-content/40 text-xs">{{ $t("cloud-chat.this-log") }}</div>
      <div class="truncate font-mono text-xs" :title="line.message">{{ line.message }}</div>
    </div>
    <button
      v-if="removable"
      type="button"
      class="btn btn-ghost btn-xs btn-square -mt-0.5 -mr-1 shrink-0"
      :aria-label="$t('cloud-chat.remove-log')"
      @click="$emit('remove')"
    >
      <mdi:close class="size-3.5" />
    </button>
  </div>
</template>

<script lang="ts" setup>
import type { ViewLogLine } from "@/composable/logs/viewContext";

const { line, removable = false } = defineProps<{ line: ViewLogLine; removable?: boolean }>();
defineEmits<{ remove: [] }>();
</script>
