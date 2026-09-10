<template>
  <section>
    <div class="mb-3 flex min-h-8 items-center gap-3">
      <button
        class="icon-btn text-base-content/60 hover:text-base-content -ml-1 flex items-center gap-2 rounded-md px-1 py-0.5 transition-colors"
        :aria-expanded="!collapsed"
        @click="collapsed = !collapsed"
      >
        <mdi:chevron-down class="size-4 transition-transform" :class="{ '-rotate-90': collapsed }" />
        <h2 class="text-xs font-semibold tracking-[0.08em] uppercase">{{ title }}</h2>
        <span
          v-if="count !== undefined"
          class="bg-base-content/10 text-base-content/70 rounded-full px-1.5 py-px text-[0.6875rem] font-semibold tabular-nums"
        >
          {{ count.toLocaleString() }}
        </span>
      </button>
      <div class="ml-auto flex items-center gap-2">
        <slot name="actions"></slot>
      </div>
    </div>
    <Transition name="collapse">
      <div v-show="!collapsed">
        <slot></slot>
      </div>
    </Transition>
  </section>
</template>

<script setup lang="ts">
defineProps<{
  title: string;
  count?: number;
}>();

const collapsed = defineModel<boolean>({ default: false });
</script>

<style scoped>
.collapse-enter-active,
.collapse-leave-active {
  transition:
    opacity 200ms cubic-bezier(0.22, 1, 0.36, 1),
    max-height 240ms cubic-bezier(0.22, 1, 0.36, 1);
  overflow: hidden;
}

.collapse-enter-from,
.collapse-leave-to {
  opacity: 0;
  max-height: 0;
}

@media (prefers-reduced-motion: reduce) {
  .collapse-enter-active,
  .collapse-leave-active {
    transition: none;
  }
}
</style>
