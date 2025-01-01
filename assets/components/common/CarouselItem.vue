<template>
  <div class="w-full min-w-full flex-shrink-0 snap-start snap-always">
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
