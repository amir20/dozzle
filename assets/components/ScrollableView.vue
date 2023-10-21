<template>
  <section :class="{ 'h-screen min-h-0': scrollable }" class="flex flex-col">
    <header
      v-if="$slots.header"
      class="sticky top-[70px] z-[2] border-b border-base-content/10 bg-base py-2 shadow-[1px_1px_2px_0_rgb(0,0,0,0.05)] md:top-0"
    >
      <slot name="header"></slot>
    </header>
    <main :data-scrolling="scrollable ? true : undefined" class="snap-y overflow-auto">
      <div class="invisible mr-28 text-right md:visible" v-show="paused">
        <scroll-progress :indeterminate="loading" :auto-hide="!loading" class="z-2 !fixed top-16" />
      </div>
      <div ref="scrollableContent">
        <slot :setLoading="setLoading"></slot>
      </div>

      <div ref="scrollObserver" class="h-px"></div>
    </main>

    <div class="mr-16 text-right">
      <transition name="fade">
        <button
          class="fixed bottom-8 rounded bg-primary p-3 text-primary-content shadow transition-colors hover:bg-primary-focus"
          :class="hasMore ? 'animate-bounce-fast bg-secondary text-secondary-content hover:bg-secondary-focus' : ''"
          @click="scrollToBottom()"
          v-show="paused"
        >
          <mdi:chevron-double-down />
        </button>
      </transition>
    </div>
  </section>
</template>

<script lang="ts" setup>
const { scrollable = false } = defineProps<{ scrollable?: boolean }>();

let paused = $ref(false);
let hasMore = $ref(false);
let loading = $ref(false);
const scrollObserver = ref<HTMLElement>();
const scrollableContent = ref<HTMLElement>();

provide("scrollingPaused", $$(paused));

const mutationObserver = new MutationObserver((e) => {
  if (!paused) {
    scrollToBottom();
  } else {
    const record = e[e.length - 1];
    const children = (record.target as HTMLElement).children;
    if (children[children.length - 1] == record.addedNodes[record.addedNodes.length - 1]) {
      hasMore = true;
    }
  }
});

const intersectionObserver = new IntersectionObserver((entries) => (paused = entries[0].intersectionRatio == 0), {
  threshold: [0, 1],
  rootMargin: "80px 0px",
});

onMounted(() => {
  mutationObserver.observe(scrollableContent.value!, { childList: true, subtree: true });
  intersectionObserver.observe(scrollObserver.value!);
});

function scrollToBottom(behavior: "auto" | "smooth" = "auto") {
  scrollObserver.value?.scrollIntoView({ behavior });
  hasMore = false;
}

function setLoading(value: boolean) {
  loading = value;
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
</style>

<style>
.splitpanes__pane {
  overflow: unset !important;
}
</style>
