<template>
  <div>
    <span v-if="modifiers.includes('shift')"><carbon:mac-shift /></span>
    <span v-if="modifiers.includes('meta')">
      <ph:command v-if="isMac" />
      <ph:control-bold v-else />
    </span>
    <kbd class="is-uppercase is-family-sans-serif">{{ char }}</kbd>
  </div>
</template>

<script lang="ts">
const isMac = /(Mac|iPhone|iPod|iPad)/i.test(navigator.userAgent);
</script>
<script lang="ts" setup>
const { modifiers = ["meta"], char } = defineProps<{ char: string; modifiers?: ("^" | "meta" | "shift")[] }>();
</script>

<style scoped lang="scss">
div {
  background-color: var(--scheme-main-ter);
  border: 1px solid var(--text-light-color);
  border-radius: 0.4em;
  padding: 0.1em 0.3em;
  color: var(--text-strong-color);
  display: flex;
  align-items: center;
  gap: 0.1em;
  line-height: 0.5;
}
</style>
