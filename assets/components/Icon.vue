<template functional>
  <svg class="icomoon" :class="['icon-' + name]">
    <use :href="'#icon-' + name"></use>
  </svg>
</template>

<script>
export default {
  props: {
    name: {
      required: true,
      type: String,
    },
  },
  name: "Icon",
};
</script>
<style lang="scss" scoped>
.icomoon {
  display: inline-block;
  width: 1em;
  height: 1em;
  stroke-width: 0;
  stroke: currentColor;
  fill: currentColor;

  .icon:not(.keep-size) & {
    width: 100%;
    height: 100%;
  }
}
</style>
