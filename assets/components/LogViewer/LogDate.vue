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
}

@media (prefers-color-scheme: dark) {
  .date {
    background-color: #262626;
    color: #258ccd;
  }
}

[data-theme="dark"] {
  .date {
    background-color: #262626;
    color: #258ccd;
  }
}

@media (prefers-color-scheme: light) {
  .date {
    background-color: #f0f0f0;
    color: #009900;
  }
}

[data-theme="light"] {
  .date {
    background-color: #f0f0f0;
    color: #009900;
  }
}
</style>
