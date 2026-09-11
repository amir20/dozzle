<template>
  <!--
    The answer for a surface that has nothing to show: a muted glyph, one line saying
    what is missing, one line saying what to do about it. It sits in the room the
    content would have filled rather than reading as a stray status line, and stays
    neutral so nothing about an empty result looks like a failure.
  -->
  <div class="flex flex-col items-center gap-4 px-4 py-16 text-center font-sans">
    <div class="bg-base-content/5 text-base-content/40 flex size-12 items-center justify-center rounded-full">
      <slot name="icon" />
    </div>
    <div class="flex flex-col gap-1.5">
      <p class="text-base font-semibold">{{ title }}</p>
      <p v-if="hint" class="text-base-content/50 text-sm">{{ hint }}</p>
    </div>
    <div v-if="$slots.actions" class="mt-1 flex flex-wrap items-center justify-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>

<script lang="ts" setup>
defineProps<{ title: string; hint?: string }>();
</script>
