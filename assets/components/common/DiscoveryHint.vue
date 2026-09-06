<template>
  <button
    ref="trigger"
    type="button"
    class="icon-btn text-base-content/25 hover:text-primary shrink-0 transition-colors"
    :title="title"
    :aria-label="title"
    :popovertarget="id"
  >
    <slot name="icon"><mdi:lightbulb-on-outline class="size-4" /></slot>
  </button>
  <div
    ref="panel"
    :id="id"
    popover
    class="rounded-box bg-base-200 border-base-content/20 fixed inset-auto m-0 max-w-[calc(100vw-1rem)] border p-3 text-xs whitespace-normal shadow-sm"
    :style="{ width: `${width}px` }"
    @beforetoggle="onBeforeToggle"
    @toggle="onToggle"
  >
    <div class="text-base-content/60 mb-2 text-[11px] tracking-wide uppercase">{{ title }}</div>
    <slot></slot>
    <div class="mt-2 flex flex-wrap items-center justify-between gap-2">
      <div><slot name="action"></slot></div>
      <div class="flex items-center gap-2">
        <a
          v-if="docs"
          :href="docs"
          target="_blank"
          rel="noopener noreferrer"
          class="link link-hover text-base-content/60"
          >{{ $t("hint.learn-more") }}</a
        >
        <button type="button" class="link link-hover text-base-content/60" @click="dismiss">
          {{ $t("hint.dismiss") }}
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
/**
 * A dismissible "did you know" popover hung off a small icon. Use it to teach an opt-in
 * feature at the point where its absence is visible, not as a general tooltip.
 *
 * The panel uses the native popover API so it renders in the top layer. Anything else gets
 * clipped by (and widens) an ancestor with `overflow`, like the container table's scroller.
 * The top layer does not escape inheritance though, so the panel resets `white-space`, which
 * the container table's cells set to `nowrap`.
 */
const {
  title,
  docs = undefined,
  width = 320,
  align = "start",
} = defineProps<{
  title: string;
  /** Documentation URL for the "Learn more" link. Omitted, the link is hidden. */
  docs?: string;
  width?: number;
  align?: "start" | "end";
}>();

const emit = defineEmits<{ dismiss: [] }>();

const id = `hint-${useId()}`;
const trigger = useTemplateRef("trigger");
const panel = useTemplateRef("panel");

const GAP = 4;
const MARGIN = 8;

// The popover is in the top layer, so it is positioned against the viewport by hand instead
// of by an anchor. CSS anchor positioning would do this, but Firefox does not have it yet.
function position() {
  if (!trigger.value || !panel.value) return;
  const anchor = trigger.value.getBoundingClientRect();
  const { offsetWidth, offsetHeight } = panel.value;

  const left = align === "end" ? anchor.right - offsetWidth : anchor.left;
  panel.value.style.left = `${Math.min(Math.max(MARGIN, left), Math.max(MARGIN, window.innerWidth - offsetWidth - MARGIN))}px`;

  const below = anchor.bottom + GAP;
  const flip = offsetHeight > 0 && below + offsetHeight > window.innerHeight - MARGIN && anchor.top > offsetHeight;
  panel.value.style.top = `${flip ? anchor.top - offsetHeight - GAP : below}px`;
}

function onBeforeToggle(event: ToggleEvent) {
  if (event.newState === "open") position();
}

function onToggle(event: ToggleEvent) {
  if (event.newState === "open") {
    // Now that it is laid out, its real height is known and it can flip above the trigger.
    position();
    window.addEventListener("scroll", position, true);
    window.addEventListener("resize", position);
  } else {
    window.removeEventListener("scroll", position, true);
    window.removeEventListener("resize", position);
  }
}

onScopeDispose(() => {
  window.removeEventListener("scroll", position, true);
  window.removeEventListener("resize", position);
});

function dismiss() {
  panel.value?.hidePopover();
  emit("dismiss");
}
</script>
