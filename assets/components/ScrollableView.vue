<template>
  <section :class="{ 'h-screen min-h-0': scrollable }" class="flex flex-col">
    <header
      v-if="$slots.header"
      class="sticky top-[70px] z-[2] border-b border-base-content/10 bg-base py-2 shadow-[1px_1px_2px_0_rgb(0,0,0,0.05)] md:top-0"
    >
      <slot name="header"></slot>
    </header>
    <main :data-scrolling="scrollable ? true : undefined" class="snap-y overflow-auto">
      <div class="invisible relative md:visible" v-show="scrollContext.paused">
        <div class="absolute right-44 top-4">
          <ScrollProgress
            :indeterminate="loadingMore"
            :auto-hide="!loadingMore"
            :progress="scrollContext.progress"
            :date="scrollContext.currentDate"
            class="!fixed z-10 min-w-40"
          />
        </div>
      </div>
      <div ref="scrollableContent">
        <slot></slot>
      </div>
      <div
        class="animate-background h-1 w-1/2 bg-gradient-radial from-primary to-transparent to-75%"
        v-show="!scrollContext.paused"
      ></div>
      <div ref="scrollObserver" class="h-px"></div>
    </main>

    <div class="mr-16 text-right">
      <transition name="fade">
        <button
          class="transition-colorsblur-xs dark btn btn-primary fixed bottom-8 rounded p-3 text-primary-content shadow"
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

const scrollContext = provideScrollContext();

const { loadingMore } = useLoggingContext();

useIntersectionObserver(scrollObserver, ([entry]) => (scrollContext.paused = entry.intersectionRatio == 0), {
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

function scrollToBottom(behavior: "auto" | "smooth" = "auto") {
  scrollObserver.value?.scrollIntoView({ behavior });
  hasMore.value = false;
}
</script>
<style scoped lang="postcss">
.fade-enter-active,
.fade-leave-active {
  @apply transition-opacity;
}

.fade-enter-from,
.fade-leave-to {
  @apply opacity-0;
}

.animate-background {
  animation: gradient-animation 3s ease-out infinite;
}

@keyframes gradient-animation {
  0%,
  100% {
    transform: translateX(0%);
  }
  50% {
    transform: translateX(100%);
  }
}
</style>

<style>
.splitpanes__pane {
  overflow: unset !important;
}
</style>
