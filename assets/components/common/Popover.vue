<template>
  <div
    ref="anchor"
    v-bind="$attrs"
    class="popover-anchor"
    @click="hover || toggle()"
    @pointerenter="onEnter"
    @pointerleave="onLeave"
  >
    <slot name="trigger" :open="isOpen"></slot>
  </div>
  <div
    ref="panel"
    popover
    class="popover-panel"
    :class="panelClass"
    @beforetoggle="onBeforeToggle"
    @toggle="onToggle"
    @pointerenter="onEnter"
    @pointerleave="onLeave"
    @click="onSelect"
  >
    <slot :close="hide" :open="isOpen"></slot>
  </div>
</template>

<script lang="ts" setup>
/**
 * A menu or panel hung off a trigger, rendered in the top layer with the native popover API.
 * Anything positioned inside the page gets clipped by an ancestor with `overflow`, which is
 * most of this app: the container table's rounded card, the log list, the sidebar.
 *
 * The trigger goes in `#trigger` and keeps its own markup, so a link stays a link. Attributes
 * land on the wrapper around it, which is what `.dropdown` used to be.
 */
defineOptions({ inheritAttrs: false });

const {
  placement = "bottom-start",
  hover = false,
  closeOnSelect = true,
  panelClass = "",
} = defineProps<{
  placement?: PopoverPlacement;
  /** Opens on pointer enter instead of click. Touch taps fire it too. */
  hover?: boolean;
  /** Close when a row is picked. Rows inside a nested <details> submenu never close. */
  closeOnSelect?: boolean;
  panelClass?: string;
}>();

const emit = defineEmits<{ open: []; close: [] }>();

const anchor = useTemplateRef<HTMLElement>("anchor");
const panel = useTemplateRef<HTMLElement>("panel");

const { isOpen, onBeforeToggle, onToggle, show, hide, toggle } = useAnchoredPopover(anchor, panel, {
  placement: () => placement,
  // Hover menus have to be reachable across the gap, so keep it tight.
  gap: () => (hover ? 2 : 4),
});

watch(isOpen, (value) => (value ? emit("open") : emit("close")));

// The panel is a sibling of the trigger, not a child, so leaving one to enter the other has
// to be bridged instead of relying on a single hover region.
let timer: ReturnType<typeof setTimeout> | undefined;
function onEnter() {
  if (!hover) return;
  clearTimeout(timer);
  show();
}
function onLeave() {
  if (!hover) return;
  clearTimeout(timer);
  timer = setTimeout(hide, 150);
}
onScopeDispose(() => clearTimeout(timer));

function onSelect(event: MouseEvent) {
  if (!closeOnSelect) return;
  const item = (event.target as HTMLElement | null)?.closest("a, button");
  // A submenu row is one step of a choice the user is still making, so it stays open.
  if (item && !item.closest("details")) hide();
}

defineExpose({ show, hide, toggle });
</script>
