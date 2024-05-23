<template>
  <div class="inline-flex items-center justify-center rounded bg-base-lighter px-2 py-[0.2em]" :size="size">
    <slot></slot>
  </div>
</template>

<script lang="ts" setup>
const { size = undefined } = defineProps<{ size?: "small" | undefined }>();
</script>

<style scoped lang="postcss">
[size="small"] {
  @apply text-xs;
}
</style>
