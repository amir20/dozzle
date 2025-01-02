<template>
  <div class="flex min-h-0 flex-col">
    <div
      ref="container"
      class="scrollbar-hide flex min-h-0 flex-1 snap-x snap-mandatory overflow-hidden overflow-x-auto overflow-y-auto overscroll-x-contain scroll-smooth"
    >
      <component v-for="(card, index) in providedCards" :key="index" :is="card" ref="cards" />
    </div>
    <div class="my-4 flex flex-none justify-center gap-2">
      <button
        v-for="(c, index) in providedCards"
        :key="c.props?.id"
        @click="scrollToItem(index)"
        :class="[
          'size-2 rounded-full transition-all duration-700',
          activeIndex === index ? 'scale-125 bg-primary' : 'bg-base-content/50 hover:bg-base-content',
        ]"
        :aria-label="c.props?.title"
        :title="c.props?.title"
      />
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
  console.log("scrollToItem");
  cards.value?.[index].$el.scrollIntoView({
    behavior: "smooth",
    inline: "start",
  });
};

const { pause, resume } = watchPausable(activeId, (v) => {
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
