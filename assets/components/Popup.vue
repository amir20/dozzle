<template>
  <slot></slot>
  <Teleport to="body">
    <Transition name="fade">
      <div
        v-show="show && (delayedShow || glopbalShow)"
        class="fixed z-50 rounded border border-base-content/10 bg-base-lighter p-4 shadow"
        ref="content"
      >
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

const onMouseEnter = (e: Event) => {
  show.value = true;
  glopbalShow.value = true;
  if (e.target && content && e.target instanceof Element) {
    const { left, top, width } = e.target.getBoundingClientRect();
    const x = left + width + 10;
    const y = top;

    content.style.left = `${x}px`;
    content.style.top = `${y}px`;
  }
};

const onMouseLeave = () => {
  show.value = false;
  glopbalShow.value = false;
};

const el = useCurrentElement();
useEventListener(() => el.value?.nextElementSibling, "mouseenter", onMouseEnter);
useEventListener(() => el.value?.nextElementSibling, "mouseleave", onMouseLeave);
</script>
