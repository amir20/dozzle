<template>
  <div class="has-text-centered is-relative host" @mouseenter="mouseOver = true" @mouseleave="mouseOver = false">
    <div class="has-border has-boxshadow">
      <stat-sparkline :data="data" @selected-point="onSelectedPoint"></stat-sparkline>
    </div>
    <div class="has-background-body-color is-top-left">
      <span class="has-text-weight-light">{{ label }}</span>
      <span class="has-text-weight-bold">
        {{ mouseOver ? selectedPoint?.value ?? selectedPoint?.y ?? statValue : statValue }}
      </span>
    </div>
  </div>
</template>

<script lang="ts" setup>
const { data, label, statValue } = defineProps<{ data: Point<unknown>[]; label: string; statValue: string | number }>();

let selectedPoint: Point<unknown> | undefined = $ref();

function onSelectedPoint(point: Point<unknown>) {
  selectedPoint = point;
}

let mouseOver = $ref(false);
</script>

<style lang="scss" scoped>
.has-border {
  border: 1px solid var(--primary-color);
  border-radius: 3px;
  padding: 1px 1px 0 1px;
  display: flex;
  overflow: hidden;
  padding-top: 0.25em;
}

.has-background-body-color {
  background-color: var(--body-background-color);
}

.host:hover span {
  color: var(--secondary-color);
}

.is-top-left {
  position: absolute;
  top: 0;
  left: 0.75em;
}
</style>
