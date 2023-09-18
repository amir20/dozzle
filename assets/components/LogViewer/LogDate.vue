<template>
  <tag size="small">
    <date-time :date="date" class="whitespace-nowrap text-[#258ccd]"></date-time>
  </tag>
</template>
<script lang="ts" setup>
defineProps<{
  date: Date;
}>();
</script>
