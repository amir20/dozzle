<template>
  <transition name="slide">
    <!-- Padded past the cloud rail so the box does not open on top of it. A
         drag sets left/top, which this does not touch. The strip itself is
         inert: it spans the window, so whatever sits under it (the topbar) has
         to stay clickable. Only the card takes the pointer. -->
    <div
      class="pointer-events-none fixed z-50 flex w-full justify-end p-2"
      v-show="showSearch"
      ref="container"
      :style="[style, { paddingRight: railOffset ? `${railOffset + 8}px` : undefined }]"
    >
      <!-- Primary on the border, not on a filled control: this is the one thing
           on screen the user is driving while it is open, and a neutral card
           floating over the stream did not say so. -->
      <div
        class="rounded-box border-primary/40 bg-base-200 focus-within:border-primary/80 pointer-events-auto flex items-center gap-1.5 border py-1.5 pr-1.5 pl-1 shadow-lg transition-colors"
        :class="{ '!border-warning/60': !isValidQuery }"
        :data-invalid="!isValidQuery || undefined"
        data-testid="search-box"
      >
        <!-- The box is draggable, but only by the grip: dragging from the input
             fights text selection. -->
        <div
          ref="handle"
          class="text-base-content/25 hover:text-base-content/50 cursor-move transition-colors"
          :title="$t('toolbar.move-search')"
        >
          <mdi:drag-vertical class="size-4" />
        </div>

        <mdi:magnify class="text-primary size-4 shrink-0" />

        <input
          class="text-base-content placeholder:text-base-content/40 w-64 min-w-0 flex-1 bg-transparent font-mono text-sm outline-none"
          type="text"
          :placeholder="$t('placeholder.find-logs')"
          ref="input"
          v-model="searchQueryFilter"
          @keyup.esc="resetSearch()"
        />

        <!-- An unparseable regex is a note on the query, not an alarm on the
             surface: the glyph carries it and the box keeps its contrast. -->
        <mdi:alert-circle-outline
          v-if="!isValidQuery"
          class="text-warning size-4 shrink-0"
          :title="$t('toolbar.invalid-regex')"
        />

        <div class="bg-base-content/10 mx-0.5 h-5 w-px shrink-0"></div>

        <button
          type="button"
          class="icon-btn inline-flex size-6 shrink-0 items-center justify-center rounded-md transition-colors"
          :class="
            inverseFilter
              ? 'bg-secondary/15 text-secondary hover:bg-secondary/25'
              : 'text-base-content/50 hover:bg-base-content/10 hover:text-base-content'
          "
          :aria-pressed="inverseFilter"
          @click="toggleInverse()"
          :title="inverseFilter ? $t('toolbar.inverse-on') : $t('toolbar.inverse-off')"
        >
          <mdi:filter-off-outline v-if="inverseFilter" class="size-4" />
          <mdi:filter-outline v-else class="size-4" />
        </button>

        <button
          type="button"
          class="icon-btn text-base-content/50 hover:bg-base-content/10 hover:text-base-content inline-flex size-6 shrink-0 items-center justify-center rounded-md transition-colors"
          @click="resetSearch()"
          :title="$t('toolbar.close-search')"
        >
          <mdi:close class="size-4" />
        </button>
      </div>
    </div>
  </transition>
</template>

<script lang="ts" setup>
const input = ref<HTMLInputElement>();
const container = ref<HTMLDivElement>();
const handle = ref<HTMLDivElement>();
const { searchQueryFilter, showSearch, resetSearch, isValidQuery, inverseFilter, toggleInverse } = useSearchFilter();
const { railOffset } = useCloudRail();

// Opening on top of the container title bar covers the stats and the toolbar
// the box is not about. It opens just under the header instead, measured rather
// than hardcoded because the header is a different height on mobile and on the
// views that carry no stats. Once the box has been dragged it stays where it
// was put.
const FALLBACK_TOP = 56;
const moved = ref(false);
const { style, position } = useDraggable(container, {
  handle,
  initialValue: { x: 0, y: FALLBACK_TOP },
  onEnd: () => (moved.value = true),
});

watch(
  showSearch,
  (open) => {
    if (!open || moved.value) return;
    nextTick(() => {
      const header = document.querySelector('[data-testid="scrollable-header"]');
      position.value = { x: 0, y: header ? Math.round(header.getBoundingClientRect().bottom) : FALLBACK_TOP };
    });
  },
  { immediate: true },
);

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

onUnmounted(() => resetSearch());
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

@media (prefers-reduced-motion: reduce) {
  .slide-enter-active,
  .slide-leave-active {
    transition: opacity 100ms linear;
  }

  .slide-enter-from,
  .slide-leave-to {
    transform: none;
  }
}
</style>
