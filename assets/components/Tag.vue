<template>
  <div class="tag" :size="size">
    <slot></slot>
  </div>
</template>

<script lang="ts" setup>
const { size = undefined } = defineProps<{ size: "small" | undefined }>();
</script>

<style scoped lang="scss">
.tag {
  background-color: var(--scheme-main-ter);
  border: 1px solid var(--border-color);
  &[size="small"] {
    font-size: 0.61rem;
  }
}
</style>
