<template>
  <div @mouseenter="onMouseEnter" @mouseleave="show = false" ref="trigger"><slot></slot></div>
  <Teleport to="body">
    <Transition name="fade">
      <div v-show="show && delayedShow" class="content" ref="content">
        <slot name="content"></slot>
      </div>
    </Transition>
  </Teleport>
</template>

<script lang="ts" setup>
import { refDebounced } from "@vueuse/core";
let show = ref(false);
let delayedShow = refDebounced(show, 1000);
let content: HTMLElement | null = $ref(null);
let trigger: HTMLElement | null = $ref(null);

function onMouseEnter(e: MouseEvent) {
  show.value = true;
  if (e.target && content) {
    const x = e.target.offsetLeft + e.target.offsetWidth + 10;
    const y = e.target.offsetTop;

    content.style.left = `${x}px`;
    content.style.top = `${y}px`;
  }
}
</script>

<style scoped>
.content {
  position: fixed;
  z-index: 9999;
  background: var(--scheme-main-ter);
  border-radius: 0.5em;
  padding: 1em;
  box-shadow: 0 0 0.5em rgba(0, 0, 0, 0.5);
  border: 1px solid var(--border-color);
}
</style>
