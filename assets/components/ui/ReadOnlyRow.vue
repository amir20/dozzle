<template>
  <!-- A read-only value is text, not a disabled input: label left, mono value right. -->
  <div class="flex items-center justify-between gap-3 p-4">
    <span class="min-w-0 text-sm">{{ label }}</span>
    <span class="text-base-content/60 shrink-0 font-mono text-xs">{{ value }}</span>
  </div>
</template>

<script lang="ts" setup>
defineProps<{ label: string; value: string }>();
</script>
