<template>
  <div class="flex min-h-0 flex-col gap-2">
    <div class="flex min-h-0 flex-1 flex-col overflow-auto overscroll-y-contain">
      <div
        ref="container"
        class="scrollbar-hide flex shrink-0 grow snap-x snap-mandatory overflow-x-auto overscroll-x-contain scroll-smooth"
      >
        <component v-for="(card, index) in providedCards" :key="index" :is="card" ref="cards" />
      </div>
    </div>
    <div class="flex flex-col gap-2">
      <h3 class="text-center text-sm font-thin">
        {{ cards?.[activeIndex].title }}
      </h3>
      <div class="flex flex-none justify-center gap-2" v-if="providedCards.length > 1">
        <button
          v-for="(c, index) in providedCards"
          :key="c.props?.id"
          @click="scrollToItem(index)"
          :class="[
            'size-2 cursor-pointer rounded-full transition-all duration-700',
            activeIndex === index ? 'bg-primary scale-125' : 'bg-base-content/50 hover:bg-base-content',
          ]"
          :aria-label="c.props?.title"
          :title="c.props?.title"
        />
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

const { pause, resume } = watchPausable(activeId, (v) => {
  if (activeId.value) {
    const index = cards.value?.map((c) => c.id).indexOf(activeId.value) ?? -1;
    if (index !== -1) {
      console.log("watching", activeId.value);
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
