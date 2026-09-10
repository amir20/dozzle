<template>
  <div class="w-full min-w-full shrink-0 snap-start snap-always p-0.5">
    <slot />
  </div>
</template>

<script lang="ts" setup>
const { id, title, description } = defineProps<{
  id: string;
  /** Short label for the tab strip. */
  title?: string;
  /** Spelled-out version, used for the tab's tooltip and accessible name. */
  description?: string;
}>();

defineExpose({
  id,
  title,
  description,
});
</script>

<style scoped></style>
