<template>
  <section :class="{ 'h-screen min-h-0': scrollable }" class="flex flex-col">
    <header
      v-if="$slots.header"
      class="border-base-content/10 bg-base-200 sticky top-[var(--mobile-nav-height)] z-20 border-b py-0.5 shadow-[1px_1px_2px_0_rgb(0,0,0,0.05)] md:top-0 md:py-2"
    >
      <slot name="header"></slot>
    </header>
    <main :data-scrolling="scrollable ? true : undefined" class="min-h-[300px] snap-y overflow-auto">
      <div ref="scrollTopObserver" class="h-px"></div>
      <div class="invisible relative md:visible" v-show="scrollContext.paused">
        <div class="absolute top-4 right-44">
          <ScrollProgress
            :indeterminate="loadingMore"
            :auto-hide="!loadingMore"
            :progress="scrollContext.progress"
            :date="scrollContext.currentDate"
            class="fixed! z-10 min-w-40"
          />
        </div>
      </div>
      <div ref="scrollableContent">
        <slot></slot>
      </div>

      <div ref="scrollObserver" class="h-px"></div>
    </main>

    <transition name="fade">
      <div
        class="border-base-content/10 fixed right-8 bottom-8 z-10 flex flex-col overflow-hidden rounded-lg border shadow-md"
        v-if="!historical"
        v-show="!atTop || scrollContext.paused"
      >
        <!-- Go to top (tonal): loads all the way back to the first line. -->
        <button
          class="btn btn-square btn-primary btn-soft rounded-none border-none shadow-none"
          :class="{ 'pointer-events-none opacity-40': atTop && !goingToTop }"
          @click="scrollToTop()"
          :aria-label="$t('button.scroll-to-top')"
          :title="$t('button.scroll-to-top')"
        >
          <span v-if="goingToTop" class="loading loading-spinner loading-sm"></span>
          <mdi:chevron-double-up v-else />
        </button>
        <!-- Go to bottom (contained): back to the live tail. -->
        <button
          class="btn btn-square btn-primary rounded-none border-none shadow-none"
          :class="{ 'animate-bounce-fast': hasMore, 'pointer-events-none opacity-40': !scrollContext.paused }"
          @click="scrollToBottom()"
          :aria-label="$t('button.scroll-to-bottom')"
          :title="$t('button.scroll-to-bottom')"
        >
          <mdi:chevron-double-down />
        </button>
      </div>
    </transition>
  </section>
</template>

<script lang="ts" setup>
const { scrollable = false } = defineProps<{ scrollable?: boolean }>();

const hasMore = ref(false);
const atTop = ref(true);
const goingToTop = ref(false);
const jumpedToTop = ref(false);
const scrollObserver = ref<HTMLElement>();
const scrollTopObserver = ref<HTMLElement>();
const scrollableContent = ref<HTMLElement>();

const scrollContext = provideScrollContext();

const { loadingMore, historical, jumpToOldest, reconnect } = useLoggingContext();
if (!historical.value) {
  useIntersectionObserver(scrollObserver, ([entry]) => (scrollContext.paused = entry.intersectionRatio == 0), {
    threshold: [0, 1],
    rootMargin: "40px 0px",
  });

  useIntersectionObserver(scrollTopObserver, ([entry]) => (atTop.value = entry.intersectionRatio != 0), {
    threshold: [0, 1],
    rootMargin: "40px 0px",
  });

  useMutationObserver(
    scrollableContent,
    (records) => {
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

async function scrollToBottom(behavior: "auto" | "smooth" = "auto") {
  // If we jumped to the oldest window, reconnect to the live tail rather than
  // scrolling within the head window.
  if (jumpedToTop.value && reconnect?.value) {
    jumpedToTop.value = false;
    scrollContext.paused = false;
    reconnect.value();
    await nextTick();
  }
  scrollObserver.value?.scrollIntoView({ behavior });
  hasMore.value = false;
}

async function scrollToTop() {
  // Jump straight to the first lines by loading only the oldest window, instead
  // of lazily loading everything in between.
  const jump = jumpToOldest?.value;
  if (jump) {
    goingToTop.value = true;
    try {
      await jump();
      jumpedToTop.value = true;
    } finally {
      goingToTop.value = false;
    }
    await nextTick();
  }
  scrollTopObserver.value?.scrollIntoView({ behavior: "auto" });
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
