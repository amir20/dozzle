<template>
  <!-- Indicators -->
  <div class="my-4 flex justify-center gap-2">
    <button
      v-for="(_, index) in providedCards"
      :key="index"
      @click="scrollToItem(index)"
      :class="[
        'h-3 w-3 rounded-full transition-all duration-700',
        activeIndex === index ? 'scale-110 bg-blue' : 'bg-base-content hover:bg-red',
      ]"
      :aria-label="`Go to slide ${index + 1}`"
    />
  </div>
  <div
    ref="container"
    class="scrollbar-hide flex snap-x snap-mandatory overflow-hidden overflow-x-auto overscroll-x-contain scroll-smooth"
  >
    <component v-for="(card, index) in providedCards" :key="index" :is="card" ref="cards" />
  </div>
</template>

<script lang="ts" setup>
import CarouselItem from "./CarouselItem.vue";
const container = useTemplateRef<HTMLDivElement>("container");
const activeIndex = ref(0);
const slots = useSlots();
const providedCards = computed(() => slots.default?.().filter((vnode) => vnode.type === CarouselItem));
const cards = useTemplateRef<InstanceType<typeof CarouselItem>[]>("cards");

const scrollToItem = (index: number) => {
  cards.value?.[index].$el.scrollIntoView({
    behavior: "smooth",
    inline: "start",
  });
};

useIntersectionObserver(
  cards as Ref<InstanceType<typeof CarouselItem>[]>,
  (entries) => {
    entries.forEach((entry) => {
      if (entry.isIntersecting) {
        const index = cards.value?.map((c) => c.$el).indexOf(entry.target);
        if (index !== undefined && index !== -1) activeIndex.value = index;
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
