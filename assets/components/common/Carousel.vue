<template>
  <div class="flex min-h-0 flex-col gap-3">
    <!-- A segmented control rather than the dots this used to carry: with at most a
         handful of sections, naming them is cheaper to read than a caption that
         only tells you where you already are. -->
    <div
      class="bg-base-content/6 flex shrink-0 gap-0.5 rounded-lg p-0.5"
      role="tablist"
      v-if="providedCards.length > 1"
    >
      <button
        v-for="(c, index) in providedCards"
        :key="c.props?.id"
        type="button"
        role="tab"
        :aria-selected="activeIndex === index"
        :title="c.props?.description ?? c.props?.title"
        :aria-label="c.props?.description ?? c.props?.title"
        @click="scrollToItem(index)"
        :class="[
          /* grow/shrink off an auto basis rather than flex-1: equal thirds wasted
             room on a short label like Hosts and truncated Services next to it. */
          'flex min-w-0 shrink grow basis-auto cursor-pointer items-center justify-center gap-1.5 rounded-md px-1.5 py-1 text-xs font-medium transition-colors',
          activeIndex === index
            ? 'bg-base-100 text-base-content shadow-sm'
            : 'text-base-content/50 hover:text-base-content/80',
        ]"
      >
        <span class="truncate">{{ c.props?.title }}</span>
      </button>
    </div>

    <div class="flex min-h-0 flex-1 flex-col overflow-auto overscroll-y-contain">
      <div
        ref="container"
        class="scrollbar-hide flex shrink-0 grow snap-x snap-mandatory overflow-x-auto overscroll-x-contain scroll-smooth"
      >
        <component v-for="(card, index) in providedCards" :key="index" :is="card" ref="cards" />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import CarouselItem from "./CarouselItem.vue";
const container = useTemplateRef<HTMLDivElement>("container");
const activeIndex = ref(0);
const activeId = defineModel<string>();
const slots = defineSlots<{ default(): VNode[] }>();
const providedCards = computed(() => slots.default().filter(({ type }) => type === CarouselItem));
const cards = useTemplateRef<InstanceType<typeof CarouselItem>[]>("cards");

const scrollToItem = (index: number) => {
  cards.value?.[index].$el.scrollIntoView({
    behavior: "smooth",
    inline: "start",
  });
};

const { pause, resume } = watchPausable(activeId, () => {
  if (activeId.value) {
    const index = cards.value?.map((c) => c.id).indexOf(activeId.value) ?? -1;
    if (index !== -1) {
      scrollToItem(index);
    }
  }
});

watchOnce(cards, () => {
  if (activeId.value) {
    const index = cards.value?.map((c) => c.id).indexOf(activeId.value) ?? -1;
    if (index !== -1) {
      scrollToItem(index);
    }
  }
});

useIntersectionObserver(
  cards as Ref<InstanceType<typeof CarouselItem>[]>,
  (entries) => {
    entries.forEach(({ isIntersecting, target }) => {
      if (isIntersecting) {
        const index = cards.value?.map((c) => c.$el).indexOf(target as HTMLDivElement) ?? -1;
        if (index !== -1) {
          pause();
          activeIndex.value = index;
          activeId.value = cards.value?.[index].id;
          nextTick(() => resume());
        }
      }
    });
  },
  {
    root: container,
    threshold: 0.5,
  },
);
</script>

<style scoped>
.scrollbar-hide {
  scrollbar-width: none;
  -ms-overflow-style: none;
}
.scrollbar-hide::-webkit-scrollbar {
  display: none;
}
</style>
