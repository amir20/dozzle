<template>
  <div class="name columns is-marginless">
    <span class="column">{{ value }}</span>
    <span class="column is-narrow" v-if="closable">
      <button class="delete is-medium" @click="$emit('close')"></button>
    </span>
  </div>
</template>

<script>
export default {
  props: {
    value: String,
    closable: {
      type: Boolean,
      default: false,
    },
  },
  name: "ContainerTitle",
};
</script>

<style lang="scss" scoped>
.name {
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  background: rgba(0, 0, 0, 0.1);
  font-weight: bold;
  font-family: monospace;
}
</style>
