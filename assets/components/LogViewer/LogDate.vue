<template>
  <Tag size="small">
    <DateTime :date="date" class="whitespace-nowrap text-blue" />
  </Tag>
</template>
<script lang="ts" setup>
defineProps<{
  date: Date;
}>();
</script>
