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
  font-weight: bold;
  font-family: monospace;

  button.delete {
    background-color: var(--scheme-main-ter);
    opacity: 0.6;
    &:after,
    &:before {
      background-color: var(--text-color);
    }

    &:hover {
      opacity: 1;
    }
  }
}
</style>
