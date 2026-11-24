<template>
  <!-- Sits right after the message it counts, not at the row's far edge. -->
  <button
    type="button"
    class="repeat-count"
    :class="{ open: fold.open }"
    :title="fold.open ? $t('tooltip.fold-repeats') : $t('tooltip.unfold-repeats', { count: fold.size })"
    @click.stop="$emit('toggle')"
  >
    ×{{ fold.size }}
  </button>
</template>

<script lang="ts" setup>
import type { FoldRun } from "@/utils/readable";

defineProps<{ fold: FoldRun }>();
defineEmits<{ toggle: [] }>();
</script>

<style scoped>
@reference "@/main.css";
.repeat-count {
  @apply bg-base-content/10 text-base-content/70 hover:bg-base-content/20 ml-2 inline-block h-fit shrink-0 cursor-pointer rounded px-1.5 align-baseline font-bold whitespace-nowrap select-none;
  text-indent: 0;

  &.open {
    @apply bg-primary/20 text-primary;
  }
}
</style>
