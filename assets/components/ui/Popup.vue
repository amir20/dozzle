<template>
  <slot></slot>
  <!-- A sibling of the row instead of a wrapper around it: these rows are <li>s in a <ul>,
       and a wrapping element would take them out of the list. -->
  <div
    ref="panel"
    popover
    class="popover-panel ring-base-content/20 bg-base-100 rounded-sm p-3 shadow-sm ring"
    @beforetoggle="onBeforeToggle"
    @toggle="onToggle"
    @pointerenter="cancelHide"
    @pointerleave="onLeave"
  >
    <!-- Mounted on first open, not with the row. The sidebar renders one of these per
         container, so on a busy host the eager version kept ~150 component trees alive
         for panels nobody had opened, each holding charts that re-derived a 300 point
         series every tick. It stays mounted afterwards: a row that has been hovered once
         tends to be hovered again, and remounting per hover would redo that work. -->
    <slot v-if="hasOpened" name="content"></slot>
  </div>
</template>

<script lang="ts">
/**
 * Shared by every popup: once one has opened, moving on to another row opens its panel at
 * once rather than waiting out the delay again.
 */
let warmUntil = 0;
</script>

<script lang="ts" setup>
/**
 * Details hung off a nav row on hover, in the top layer via the native popover API so the
 * sidebar's scroller cannot clip it. Only one auto popover stays open at a time, so moving
 * between rows closes the previous panel for free.
 */
const panel = useTemplateRef<HTMLElement>("panel");
// The trigger is whatever the slot rendered right before the panel, which is the row.
const anchor = computed(() => (panel.value?.previousElementSibling as HTMLElement | null) ?? null);

const { isOpen, onBeforeToggle, onToggle, show, hide } = useAnchoredPopover(anchor, panel, {
  placement: () => "right-start",
  gap: () => 10,
});

// Long enough that the panel does not flash while the pointer crosses the list on its way
// somewhere else.
const OPEN_DELAY = 1000;
// Hiding is delayed so the pointer can cross the gap onto the panel, which is hoverable
// itself, without the popup vanishing on the way.
const HIDE_DELAY = 200;

let timer: ReturnType<typeof setTimeout> | undefined;

const cancelHide = () => clearTimeout(timer);

const hasOpened = ref(false);

// The panel is measured and placed immediately after showPopover(), so its contents have
// to be in the DOM before that runs or the first frame is positioned against an empty box.
const open = async () => {
  if (!hasOpened.value) {
    hasOpened.value = true;
    await nextTick();
  }
  show();
};

const onEnter = () => {
  clearTimeout(timer);
  if (performance.now() < warmUntil) open();
  else timer = setTimeout(open, OPEN_DELAY);
};

const onLeave = () => {
  clearTimeout(timer);
  if (isOpen.value) warmUntil = performance.now() + OPEN_DELAY;
  timer = setTimeout(hide, HIDE_DELAY);
};

useEventListener(anchor, "pointerenter", onEnter);
useEventListener(anchor, "pointerleave", onLeave);

onScopeDispose(() => clearTimeout(timer));
</script>
