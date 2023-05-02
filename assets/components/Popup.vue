<template>
  <div @mouseenter="onMouseEnter" @mouseleave="onMouseLeave" ref="trigger"><slot></slot></div>
  <Teleport to="body">
    <Transition name="fade">
      <div v-show="show && (delayedShow || glopbalShow)" class="content" ref="content">
        <slot name="content"></slot>
      </div>
    </Transition>
  </Teleport>
</template>

<script lang="ts" setup>
import { globalShowPopup } from "@/composables/popup";

let glopbalShow = globalShowPopup();
let show = ref(glopbalShow.value);
let delayedShow = refDebounced(show, 1000);

let content: HTMLElement | null = $ref(null);
let trigger: HTMLElement | null = $ref(null);

function onMouseEnter(e: MouseEvent) {
  show.value = true;
  glopbalShow.value = true;
  if (e.target && content && e.target instanceof Element) {
    const { left, top, width } = e.target.getBoundingClientRect();
    const x = left + width + 10;
    const y = top;

    content.style.left = `${x}px`;
    content.style.top = `${y}px`;
  }
}

function onMouseLeave() {
  show.value = false;
  glopbalShow.value = false;
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
