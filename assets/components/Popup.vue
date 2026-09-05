<template>
  <slot></slot>
  <teleport to="body">
    <transition name="fade">
      <div
        v-show="show && (delayedShow || globalShow)"
        class="ring-base-content/20 bg-base-100 fixed z-50 rounded-sm p-3 shadow-sm ring"
        ref="content"
      >
        <slot name="content"></slot>
      </div>
    </transition>
  </teleport>
</template>

<script lang="ts" setup>
import { activePopup, globalShowPopup } from "@/composable/popup";

const globalShow = globalShowPopup();
const active = activePopup();
const id = Symbol();
const show = ref(globalShow.value);
const delayedShow = refDebounced(show, 1000);
const content = ref<HTMLElement>();

// Hiding is delayed so the pointer can cross the gap between the trigger and the
// content, which is hoverable itself, without the popup vanishing on the way.
const HIDE_DELAY = 200;
let hideTimer: ReturnType<typeof setTimeout> | undefined;

const cancelHide = () => {
  clearTimeout(hideTimer);
  hideTimer = undefined;
};

const onMouseEnter = (e: Event) => {
  cancelHide();
  active.value = id;
  show.value = true;
  globalShow.value = true;

  if (content.value && e.target instanceof HTMLElement) {
    const { left, top, width } = e.target.getBoundingClientRect();
    const x = left + width + 10;
    const y = top;

    content.value.style.left = `${x}px`;
    content.value.style.top = `${y}px`;
  }
};

const onMouseLeave = () => {
  cancelHide();
  hideTimer = setTimeout(() => {
    if (active.value === id) active.value = null;
    show.value = false;
    globalShow.value = false;
    hideTimer = undefined;
  }, HIDE_DELAY);
};

// Another popup took over: drop this one now instead of waiting out the grace period.
watch(active, (current) => {
  if (current === id || !show.value) return;
  cancelHide();
  show.value = false;
});

onScopeDispose(cancelHide);

const el: Ref<HTMLElement> = useCurrentElement();
useEventListener(() => el.value?.nextElementSibling, "mouseenter", onMouseEnter);
useEventListener(() => el.value?.nextElementSibling, "mouseleave", onMouseLeave);
useEventListener(content, "mouseenter", cancelHide);
useEventListener(content, "mouseleave", onMouseLeave);
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  @apply transition-opacity;
}

.fade-enter-from,
.fade-leave-to {
  @apply opacity-0;
}
</style>
