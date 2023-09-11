<template>
  <div class="rounded bg-scheme-main-ter px-1 py-[0.2em]" :size="size" :class="sizes[size]">
    <slot></slot>
  </div>
</template>

<script lang="ts" setup>
const { size = undefined } = defineProps<{ size?: "small" | undefined }>();
const sizes = {
  small: "text-xs",
  medium: "text-sm",
  large: "text-base",
};
</script>

<style scoped></style>
