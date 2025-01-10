<template>
  <div class="w-full min-w-full shrink-0 snap-start snap-always p-0.5">
    <slot />
  </div>
</template>

<script lang="ts" setup>
const { id, title } = defineProps<{ id: string; title?: string }>();

defineExpose({
  id,
  title,
});
</script>

<style scoped></style>
