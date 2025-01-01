<template>
  <div class="h-64 w-full min-w-full flex-shrink-0 snap-start snap-always">
    {{ title }}
    <slot />
  </div>
</template>

<script lang="ts" setup>
const { title } = defineProps<{ title: string }>();

defineExpose({
  title,
});
</script>

<style scoped></style>
