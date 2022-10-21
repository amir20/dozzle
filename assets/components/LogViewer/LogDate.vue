<template>
  <relative-time :date="date" class="date"></relative-time>
</template>
<script lang="ts" setup>
defineProps<{
  date: Date;
}>();
</script>

<style lang="scss" scoped>
.date {
  padding-left: 5px;
  padding-right: 5px;
  border-radius: 3px;
  white-space: nowrap;
  background-color: var(--scheme-main-ter);
  color: #258ccd;
}
</style>
