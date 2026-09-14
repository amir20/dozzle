<template>
  <div class="flex items-center gap-3 p-4">
    <div class="bg-success/10 text-success shrink-0 rounded-full p-1">
      <mdi:check class="size-3.5" />
    </div>
    <span class="min-w-0 flex-1 text-sm">{{ label }}</span>
    <span v-if="value" class="text-base-content/60 shrink-0 font-mono text-xs">{{ value }}</span>
  </div>
</template>

<script lang="ts" setup>
defineProps<{ label: string; value?: string }>();
</script>
