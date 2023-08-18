<template>
  <span class="icon-text">
    <span class="icon mx-0" v-if="modifiers.includes('shift')">
      <carbon:mac-shift />
    </span>
    <span class="icon mx-0" v-if="modifiers.includes('meta')">
      <ph:command v-if="isMac" />
      <ph:control-bold v-else />
    </span>
    <kbd class="is-uppercase is-family-sans-serif ml-1">{{ char }}</kbd>
  </span>
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
