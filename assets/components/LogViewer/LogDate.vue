<template>
  <tag size="small">
    <date-time :date="date" class="whitespace-nowrap text-blue"></date-time>
  </tag>
</template>
<script lang="ts" setup>
defineProps<{
  date: Date;
}>();
</script>
