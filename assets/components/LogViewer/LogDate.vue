<template>
  <Tag size="small" class="items-start!">
    <DateTime :date="date" class="text-blue whitespace-nowrap" />
  </Tag>
</template>
<script lang="ts" setup>
defineProps<{
  date: Date;
}>();
</script>
