<template>
  <section :class="{ 'h-screen min-h-0': scrollable }" class="flex flex-col">
    <header
      v-if="$slots.header"
      ref="scrollableHeader"
      data-testid="scrollable-header"
      class="border-base-content/10 bg-base-200 sticky top-[var(--mobile-nav-offset)] z-20 border-b py-0.5 shadow-[1px_1px_2px_0_rgb(0,0,0,0.05)] md:top-0 md:py-2"
    >
      <slot name="header"></slot>
    </header>
    <!-- Fixed and centred on the measured log column: `main` is only the
         scroller when `scrollable` is set, so sticky would fail on the pages
         where the document scrolls instead, and a split pane means the column
         is not the window either. -->
    <div
      class="pointer-events-none fixed z-20 flex -translate-x-1/2 justify-center max-md:hidden"
      :style="readoutStyle"
    >
      <ScrollProgress
        :indeterminate="loadingMore"
        :show="scrollContext.paused && (isScrolling || loadingMore) && (scrollContext.available || loadingMore)"
        :available="scrollContext.available"
        :progress="scrollContext.progress"
        :date="scrollContext.currentDate"
      />
    </div>
    <main ref="scrollableMain" :data-scrolling="scrollable ? true : undefined" class="min-h-75 snap-y overflow-auto">
      <!-- The find box floats over the top of this column, so the list starts below
           it while it is open and sitting where it opened. -->
      <div
        ref="scrollableContent"
        :style="{ paddingTop: searchOverlayHeight ? `${searchOverlayHeight}px` : undefined }"
      >
        <slot></slot>
      </div>

      <div ref="scrollObserver" class="h-px"></div>
    </main>

    <div class="mr-16 text-right" v-if="!historical">
      <transition name="fade">
        <button
          class="icon-btn btn btn-primary text-primary-content fixed bottom-8 rounded-sm p-3 shadow-sm transition-colors"
          :class="hasMore ? 'btn-secondary animate-bounce-fast text-secondary-content' : ''"
          @click="scrollToBottom()"
          v-show="scrollContext.paused"
        >
          <mdi:chevron-double-down />
        </button>
      </transition>
    </div>
  </section>
</template>

<script lang="ts" setup>
const { scrollable = false, ownsViewContext = true } = defineProps<{
  scrollable?: boolean;
  /** False for a pinned column, which is a second view of some other
   *  container: the assistant should follow the primary viewer, not it. */
  ownsViewContext?: boolean;
}>();

const hasMore = ref(false);
const scrollObserver = ref<HTMLElement>();
const scrollableContent = ref<HTMLElement>();
const scrollableMain = ref<HTMLElement>();
const scrollableHeader = ref<HTMLElement>();

// Overlay-scrollbar behaviour: the readout shows while the user is moving and
// for a second after they stop, then fades out, so a paused stream is never
// left without its position but an opaque panel is never parked over the logs
// either. Both scrollers are watched because only one of them moves,
// depending on whether this view owns its scrolling.
//
// The hold is ours rather than `useScroll`'s `isScrolling`: that flag also
// listens for the native `scrollend` event, which is dispatched undebounced
// and so ignores `idle` entirely. In a browser that fires `scrollend` (Chrome)
// it drops the instant the gesture settles and takes the readout with it.
// Watching the offset and letting the flag expire on its own keeps the timing
// where this component can set it.
const SCROLL_HOLD = 1000;
const { y: yMain } = useScroll(scrollableMain);
const { y: yWindow } = useScroll(window);
const isScrolling = refAutoReset(false, SCROLL_HOLD);
watch([yMain, yWindow], () => (isScrolling.value = true));

const mainBounds = useElementBounding(scrollableMain);
const headerBounds = useElementBounding(scrollableHeader);
const readoutStyle = computed(() => ({
  left: `${mainBounds.left.value + mainBounds.width.value / 2}px`,
  // The header is sticky, so its bottom edge is where the logs actually start
  // once the page has scrolled past the top of the column.
  top: `${Math.max(headerBounds.bottom.value, mainBounds.top.value) + 8}px`,
}));

const scrollContext = provideScrollContext();

const { loadingMore, historical } = useLoggingContext();
provideViewContextOwner(ownsViewContext);
if (ownsViewContext) publishViewContext(buildViewContext(scrollContext));

const { isSearching, searchOverlayHeight } = useSearchFilter();
if (!historical.value) {
  useIntersectionObserver(scrollObserver, ([entry]) => (scrollContext.paused = entry.intersectionRatio == 0), {
    threshold: [0, 1],
    rootMargin: "40px 0px",
  });

  useMutationObserver(
    scrollableContent,
    (records) => {
      if (isMobile.value && isSearching.value) return;
      if (!scrollContext.paused) {
        scrollToBottom();
      } else {
        const record = records[records.length - 1];
        const children = (record.target as HTMLElement).children;
        if (children[children.length - 1] == record.addedNodes[record.addedNodes.length - 1]) {
          hasMore.value = true;
        }
      }
    },
    { childList: true, subtree: true },
  );
}

function scrollToBottom(behavior: "auto" | "smooth" = "auto") {
  scrollObserver.value?.scrollIntoView({ behavior });
  hasMore.value = false;
}
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

<style>
.splitpanes__pane {
  overflow: unset !important;
}
</style>
