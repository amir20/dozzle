<template>
  <!-- A read-only value is text, not a disabled input: label left, mono value right,
       with an optional status glyph leading. -->
  <div class="flex items-center gap-3 p-4">
    <slot name="icon" />
    <span class="min-w-0 flex-1 text-sm">{{ label }}</span>
    <span v-if="value" class="text-base-content/60 shrink-0 font-mono text-xs">{{ value }}</span>
  </div>
</template>

<script lang="ts" setup>
defineProps<{ label: string; value?: string }>();
</script>
