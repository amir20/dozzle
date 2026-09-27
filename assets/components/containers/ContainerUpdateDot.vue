<template>
  <!-- Sits beside AlertDot and follows it: a 6px mark on the name rather than
       a column, so a table where everything is current is unchanged. The ring
       is what tells it apart from an alert, which is solid. -->
  <button
    v-if="hasUpdate(container)"
    type="button"
    class="update-dot text-info"
    :title="$t('toolbar.update-available')"
    :aria-label="$t('toolbar.update-available')"
    @click.stop="showDrawer(ContainerUpdatesDrawer, { focus: container.id }, 'lg')"
  ></button>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import ContainerUpdatesDrawer from "./ContainerUpdatesDrawer.vue";

defineProps<{ container: Container }>();

const { hasUpdate } = useImageUpdates();
const showDrawer = useDrawer();
</script>

<style scoped>
@reference "@/main.css";

.update-dot {
  @apply relative size-2 shrink-0 rounded-full border-[1.5px] border-current transition-[box-shadow,transform];
}

/* The same invisible ~24px target AlertDot uses. */
.update-dot::after {
  content: "";
  @apply absolute -inset-2;
}

.update-dot:hover,
.update-dot:focus-visible {
  box-shadow: 0 0 0 3px color-mix(in oklab, currentColor 30%, transparent);
}

@media (prefers-reduced-motion: no-preference) {
  .update-dot:hover {
    transform: scale(1.2);
  }
}
</style>
