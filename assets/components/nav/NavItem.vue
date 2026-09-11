<template>
  <li>
    <component
      :is="tag"
      v-bind="{ ...bindings, ...$attrs }"
      class="nav-item group/nav-item"
      :class="{ 'is-active': active, 'is-muted': muted }"
    >
      <span class="flex size-5 shrink-0 items-center justify-center" v-if="$slots.icon">
        <slot name="icon" />
      </span>
      <span class="min-w-0 flex-1 truncate">
        <slot>{{ label }}</slot>
      </span>
      <span class="flex shrink-0 items-center gap-1" v-if="$slots.trailing">
        <slot name="trailing" />
      </span>
    </component>
  </li>
</template>

<script lang="ts" setup>
import { RouterLink, type RouteLocationRaw } from "vue-router";

// Attributes land on the link/button rather than the wrapping <li>, so callers can
// hang click handlers, titles and state classes straight on the interactive row.
defineOptions({ inheritAttrs: false });

const {
  to,
  active = false,
  muted = false,
} = defineProps<{
  /** Renders a router-link when given, a plain button otherwise. */
  to?: RouteLocationRaw;
  label?: string;
  /** Only consulted for the button form; a router-link tracks the route itself. */
  active?: boolean;
  /** Dimmed and unclickable, e.g. an offline host. */
  muted?: boolean;
}>();

const tag = computed(() => (to ? RouterLink : "button"));
const bindings = computed(() => (to ? { to, activeClass: "is-active" } : { type: "button" }));
</script>

<style scoped>
@reference "@/main.css";

.nav-item {
  @apply text-base-content/85 hover:text-base-content hover:bg-base-content/8 relative flex h-8 w-full cursor-pointer items-center gap-2 rounded-md px-2 text-left text-[0.9375rem] transition-colors;
}

/* Tinted rather than filled: a solid primary block on the selected row shouted
 * over the container icon and status dot sitting inside it. */
.nav-item.is-active {
  @apply bg-primary/15 text-primary font-medium;
}

/* A merged view has no single active row, so every container feeding the stream
 * carries the active row's tint in the accent colour instead. */
.nav-item.is-merged {
  @apply text-secondary bg-secondary/12 font-medium;
}

.nav-item.is-muted {
  @apply text-base-content/40 pointer-events-none;
}
</style>
