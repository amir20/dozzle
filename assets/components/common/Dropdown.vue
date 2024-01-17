<template>
  <div class="dropdown">
    <label tabindex="0" class="btn btn-circle btn-sm" @mousedown="checkAndCloseDropDown($event)" @blur="closed()">
      <slot name="trigger"></slot>
    </label>
    <div
      tabindex="0"
      class="dropdown-content z-50 mt-1 min-w-52 rounded-box border border-base-content/20 bg-base p-2 shadow"
    >
      <slot name="content"></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
const closed = defineEmit();
function checkAndCloseDropDown(e: MouseEvent) {
  const target = e.currentTarget as HTMLElement;
  if (target?.matches(":focus")) {
    setTimeout(() => target.blur(), 0);
  }
}
</script>

<style scoped></style>
