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

    <div class="fixed right-16 bottom-8 flex flex-row items-center gap-2" v-if="!historical">
      <transition name="fade">
        <button
          class="btn btn-primary text-primary-content rounded-sm p-3 shadow-sm transition-colors"
          @click="scrollToTop()"
          v-show="!atTop"
          :aria-label="$t('button.scroll-to-top')"
          :title="$t('button.scroll-to-top')"
        >
          <mdi:chevron-double-up />
        </button>
      </transition>
      <transition name="fade">
        <button
          class="btn btn-primary text-primary-content rounded-sm p-3 shadow-sm transition-colors"
          :class="hasMore ? 'btn-secondary animate-bounce-fast text-secondary-content' : ''"
          @click="scrollToBottom()"
          v-show="scrollContext.paused"
          :aria-label="$t('button.scroll-to-bottom')"
          :title="$t('button.scroll-to-bottom')"
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
const atTop = ref(true);
const scrollObserver = ref<HTMLElement>();
const scrollTopObserver = ref<HTMLElement>();
const scrollableContent = ref<HTMLElement>();

const scrollContext = provideScrollContext();

const { loadingMore, historical } = useLoggingContext();
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

function scrollToBottom(behavior: "auto" | "smooth" = "auto") {
  scrollObserver.value?.scrollIntoView({ behavior });
  hasMore.value = false;
}

function scrollToTop(behavior: "auto" | "smooth" = "auto") {
  scrollTopObserver.value?.scrollIntoView({ behavior });
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
