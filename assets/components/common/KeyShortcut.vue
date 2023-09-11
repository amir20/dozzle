<template>
  <div class="inline-flex items-center">
    <span v-if="modifiers.includes('shift')">
      <carbon:mac-shift />
    </span>
    <span v-if="modifiers.includes('meta')">
      <ph:command v-if="isMac" />
      <ph:control-bold v-else />
    </span>
    <kbd class="uppercase">{{ char }}</kbd>
  </div>
</template>

<script lang="ts">
const isMac = /(Mac|iPhone|iPod|iPad)/i.test(navigator.userAgent);
</script>
<script lang="ts" setup>
const { modifiers = ["meta"], char } = defineProps<{ char: string; modifiers?: ("^" | "meta" | "shift")[] }>();
</script>

<style scoped lang="scss">
.icon {
  width: unset;
}
</style>
