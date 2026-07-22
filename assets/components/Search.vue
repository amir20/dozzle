<template>
  <transition name="slide">
    <div
      class="fixed z-[15] flex w-full justify-end p-2"
      :class="{ 'reset-anim': resetting }"
      v-show="showSearch"
      ref="container"
      :style="style"
    >
      <div
        class="input input-primary flex items-center shadow-lg transition-transform duration-300 ease-out"
        :class="!isValidQuery ? 'input-warning' : ''"
        :style="{ transform: nudged ? 'translateX(-14rem)' : '' }"
      >
        <div class="tooltip tooltip-bottom" :data-tip="$t('toolbar.reset-position')">
          <button
            type="button"
            class="relative grid size-6 shrink-0 cursor-pointer place-items-center rounded-full select-none"
            :aria-label="$t('toolbar.reset-position')"
            @pointerdown="startHold"
            @pointerup="cancelHold"
            @pointerleave="cancelHold"
            @pointercancel="cancelHold"
            @contextmenu.prevent
          >
            <mdi:magnify />
            <svg
              v-show="holding"
              class="text-primary pointer-events-none absolute inset-0 size-full -rotate-90"
              viewBox="0 0 36 36"
              fill="none"
            >
              <circle
                cx="18"
                cy="18"
                r="16"
                stroke="currentColor"
                stroke-width="3"
                stroke-linecap="round"
                stroke-dasharray="100.53"
                :class="{ 'ring-fill': holding }"
                style="stroke-dashoffset: 100.53"
              />
            </svg>
          </button>
        </div>
        <input
          class="input input-ghost w-72 min-w-0 flex-1"
          type="text"
          :placeholder="$t('toolbar.search-placeholder')"
          :title="$t('toolbar.search')"
          :aria-label="$t('toolbar.search')"
          ref="input"
          v-model="searchQueryFilter"
          @keyup.esc="resetSearch()"
        />
        <div
          class="tooltip tooltip-bottom tooltip-end"
          :data-tip="inverseFilter ? $t('toolbar.inverse-on') : $t('toolbar.inverse-off')"
        >
          <button
            class="btn btn-circle btn-xs"
            :class="inverseFilter ? 'btn-error hover:bg-error!' : 'btn-ghost'"
            @click="toggleInverse()"
            data-testid="inverse-filter"
            :aria-label="inverseFilter ? $t('toolbar.inverse-on') : $t('toolbar.inverse-off')"
          >
            <mdi:filter-off-outline v-if="inverseFilter" />
            <mdi:filter-outline v-else />
          </button>
        </div>
        <div class="tooltip tooltip-bottom tooltip-end" :data-tip="$t('toolbar.close')">
          <a class="btn btn-circle btn-xs btn-ghost" @click="resetSearch()" :aria-label="$t('toolbar.close')">
            <mdi:close />
          </a>
        </div>
      </div>
    </div>
  </transition>
</template>

<script lang="ts" setup>
const HOLD_MS = 1500;
const DEFAULT_POSITION = { x: 0, y: 64 };

const input = ref<HTMLInputElement>();
const container = ref<HTMLDivElement>();
const { searchQueryFilter, showSearch, resetSearch, isValidQuery, inverseFilter, toggleInverse, actionsMenuOpen } =
  useSearchFilter();

// Whether the user has dragged the bar. Once moved, we stop auto-repositioning
// it and leave it where they put it (until a long-press reset).
const manuallyMoved = ref(false);
// Whether the bar is nudged aside to make room for the actions menu.
const nudged = ref(false);
// Enables the position transition while snapping back to the default spot.
const resetting = ref(false);

// Start below the header so the field doesn't cover the stats and actions menu.
const { x, y, style } = useDraggable(container, {
  initialValue: { ...DEFAULT_POSITION },
  onMove: () => {
    if (!manuallyMoved.value) {
      manuallyMoved.value = true;
      nudged.value = false;
    }
  },
});

// Sit just below the header. Its height differs between the desktop and mobile
// layouts (mobile adds a top nav), so measure it rather than hard-code a value,
// otherwise the bar can land behind the status bar on mobile.
function defaultY() {
  // The bar's own p-2 wrapper already adds the gap below the header, matching
  // the gap to the right edge, so don't add extra offset here.
  const header = document.querySelector("header");
  return header ? Math.round(header.getBoundingClientRect().bottom) : DEFAULT_POSITION.y;
}

// Reposition below the header each time the bar opens, unless the user has
// dragged it somewhere.
watch(showSearch, (visible) => {
  if (visible && !manuallyMoved.value) {
    x.value = DEFAULT_POSITION.x;
    y.value = defaultY();
  }
});

// Move aside when the actions menu opens over the bar, then slide back when it
// closes, unless the user has taken manual control of the position.
watch([actionsMenuOpen, showSearch], ([menuOpen, visible]) => {
  if (menuOpen && visible && !manuallyMoved.value) {
    nudged.value = true;
  } else if (!menuOpen) {
    nudged.value = false;
  }
});

// Long-press the search icon (macOS style) to snap the bar back to its default
// position. A ring around the icon fills over the hold duration.
const holding = ref(false);
let holdTimer: ReturnType<typeof setTimeout> | undefined;

function startHold(e: PointerEvent) {
  if (e.pointerType === "mouse" && e.button !== 0) return;
  // Don't let useDraggable treat the press as the start of a drag.
  e.stopPropagation();
  e.preventDefault();
  holding.value = true;
  holdTimer = setTimeout(() => {
    holdTimer = undefined;
    holding.value = false;
    resetPosition();
  }, HOLD_MS);
}

function cancelHold() {
  if (holdTimer) {
    clearTimeout(holdTimer);
    holdTimer = undefined;
  }
  holding.value = false;
}

function resetPosition() {
  resetting.value = true;
  nudged.value = false;
  manuallyMoved.value = false;
  x.value = DEFAULT_POSITION.x;
  y.value = defaultY();
  setTimeout(() => (resetting.value = false), 450);
}

onKeyStroke("f", (e) => {
  if (!search.value) return;
  if ((e.ctrlKey || e.metaKey) && !e.shiftKey) {
    showSearch.value = true;
    nextTick(() => input.value?.focus() || input.value?.select());
    e.preventDefault();
  }
});

onMounted(() => {
  onKeyStroke(
    "f",
    (e) => {
      if (e.ctrlKey || e.metaKey) {
        e.stopPropagation();
        resetSearch();
      }
    },
    { target: input.value },
  );
});

onUnmounted(() => {
  cancelHold();
  resetSearch();
});
</script>

<style scoped>
.slide-enter-active,
.slide-leave-active {
  transition: all 200ms cubic-bezier(0.175, 0.885, 0.32, 1.275);
}

.slide-enter-from,
.slide-leave-to {
  transform: translateY(-150px);
  opacity: 0;
}

/* Animate left/top only while snapping back to the default position. */
.reset-anim {
  transition:
    left 0.4s cubic-bezier(0.2, 0, 0, 1),
    top 0.4s cubic-bezier(0.2, 0, 0, 1);
}

/* Fill the ring around the search icon over the hold duration. */
/* Duration must track HOLD_MS so the ring finishes exactly as the hold fires. */
.ring-fill {
  animation: ring-fill 1500ms linear forwards;
}

@keyframes ring-fill {
  from {
    stroke-dashoffset: 100.53;
  }
  to {
    stroke-dashoffset: 0;
  }
}
</style>
