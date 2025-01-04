<template>
  <div
    class="tag bg-base-100 inline-flex items-center justify-center rounded-sm px-2 py-[0.2em] [[size='small']]:text-sm"
    :size="size"
  >
    <slot></slot>
  </div>
</template>

<script lang="ts" setup>
const { size = undefined } = defineProps<{ size?: "small" | undefined }>();
</script>
