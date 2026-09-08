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
    <main
      ref="scrollableMain"
      :data-scrolling="scrollable ? true : undefined"
      class="min-h-[300px] snap-y overflow-auto"
    >
      <div ref="scrollableContent">
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
const { scrollable = false } = defineProps<{ scrollable?: boolean }>();

const hasMore = ref(false);
const scrollObserver = ref<HTMLElement>();
const scrollableContent = ref<HTMLElement>();
const scrollableMain = ref<HTMLElement>();
const scrollableHeader = ref<HTMLElement>();

// Overlay-scrollbar behaviour: the readout is at full strength while the user
// is moving and settles back to a dim hint a moment after they stop, so a
// paused stream is never left without its position but reading it is never
// fought either. Both scrollers are watched because only one of them moves,
// depending on whether this view owns its scrolling.
const { isScrolling: isScrollingMain } = useScroll(scrollableMain, { idle: 1200 });
const { isScrolling: isScrollingWindow } = useScroll(window, { idle: 1200 });
const isScrolling = computed(() => isScrollingMain.value || isScrollingWindow.value);

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
const { isSearching } = useSearchFilter();
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
