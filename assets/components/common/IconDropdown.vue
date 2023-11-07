<template>
  <div class="dropdown">
    <label tabindex="0" class="btn btn-circle btn-sm">
      <slot name="trigger"></slot>
    </label>
    <div
      tabindex="0"
      class="dropdown-content rounded-box z-50 mt-1 w-52 border border-base-content/20 bg-base p-2 shadow"
    >
      <slot name="content"></slot>
    </div>
  </div>
</template>

<script setup lang="ts"></script>

<style scoped></style>
