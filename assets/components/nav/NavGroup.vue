<template>
  <li>
    <div class="group/header flex h-7 items-center gap-1 rounded-md pr-0.5 pl-1">
      <button type="button" class="nav-group-toggle" :aria-expanded="open" @click="open = !open">
        <mdi:chevron-right class="size-3.5 shrink-0 transition-transform duration-150" :class="{ 'rotate-90': open }" />
        <component :is="icon" v-if="icon" class="size-3.5 shrink-0" />
        <span class="truncate">{{ label }}</span>
        <span class="shrink-0 font-normal tabular-nums opacity-60" v-if="count !== undefined">{{ count }}</span>
      </button>

      <!-- Row actions stay out of the way on a pointer device and stay put on
           touch, where there is no hover to reveal them with. -->
      <div
        class="flex shrink-0 items-center gap-0.5 focus-within:opacity-100 md:opacity-0 md:group-hover/header:opacity-100"
        v-if="$slots.actions"
      >
        <slot name="actions" />
      </div>
    </div>

    <ul class="border-base-content/10 mt-0.5 ml-2.5 space-y-px border-l pl-1" v-show="open">
      <slot />
    </ul>
  </li>
</template>

<script lang="ts" setup>
import type { Component } from "vue";

const { label, count, icon } = defineProps<{
  label: string;
  count?: number;
  icon?: Component;
}>();

const open = defineModel<boolean>("open", { default: true });
</script>

<style scoped>
@reference "@/main.css";

/* Deliberately quieter than an item row: the group is scaffolding, the
 * containers under it are the content. Sentence case rather than the usual
 * uppercase micro-label — a namespace like `docker-compose-project` is already
 * long, and uppercase plus letter-spacing truncated it in a 15%-wide sidebar. */
.nav-group-toggle {
  @apply text-base-content/55 hover:text-base-content/90 flex min-w-0 flex-1 cursor-pointer items-center gap-1.5 text-left text-xs font-semibold transition-colors;
}
</style>
