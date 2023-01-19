<template>
  <div :class="level"></div>
</template>
<script lang="ts" setup>
defineProps<{
  level: string;
}>();
</script>

<style lang="scss" scoped>
div {
  display: inline-block;
  width: 0.7em;
  height: 0.7em;
  border-radius: 50%;

  &.debug {
    background-color: #00b5ad;
  }

  &.info {
    background-color: #00b5ad;
  }

  &.error {
    background-color: #f44336;
  }

  &.warning {
    background-color: #ff9800;
  }
}
</style>
