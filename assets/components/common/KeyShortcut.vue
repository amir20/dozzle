<template>
  <div class="inline-flex items-center">
    <template v-if="modifiers.includes('shift')">
      <carbon:mac-shift />
    </template>
    <template v-if="modifiers.includes('meta')">
      <ph:command v-if="isMac" class="size-4" />
      <ph:control-bold v-else class="size-4" />
    </template>
    <kbd class="uppercase">{{ char }}</kbd>
  </div>
</template>

<script lang="ts">
const isMac = /(Mac|iPhone|iPod|iPad)/i.test(navigator.userAgent);
</script>
<script lang="ts" setup>
const { modifiers = ["meta"], char } = defineProps<{ char: string; modifiers?: ("^" | "meta" | "shift")[] }>();
</script>
