<template>
  <div class="columns is-marginless has-text-weight-bold is-family-monospace">
    <span class="column">{{ value }}</span>
  </div>
</template>

<script>
export default {
  props: {
    value: String,
  },
  name: "ContainerTitle",
};
</script>

<style lang="scss" scoped></style>
