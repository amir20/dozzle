<template>
  <div class="columns is-marginless has-text-weight-bold is-family-monospace">
    <span class="column is-ellipsis"
      >{{ container.name }} <span class="tag is-dark">{{ container.image }}</span></span
    >
  </div>
</template>

<script>
export default {
  props: {
    container: Object,
  },
  name: "ContainerTitle",
};
</script>

<style lang="scss" scoped></style>
