<template>
  <!--
    A strip of icons on the right edge: how people find Cloud at all, and the
    edge of the drawer its panels open in. While a panel is open the strip
    takes the drawer's fill and drops its border, so the two read as one
    surface, and the icons are the drawer's tabs.

    Fixed rather than a splitpanes column, and the layout pads the page by
    exactly its width, so the strip never sits over the stream. It is mounted
    only where cloud is linked, so the mark at the top is a statement of where
    the answers come from and never an advert. It opens the account and usage.
  -->
  <nav
    class="rail fixed inset-y-0 right-0 z-40 flex flex-col items-center gap-1 border-l py-3"
    :class="panel ? 'bg-base-100 border-transparent' : 'border-base-content/10 bg-base-200/40'"
    :style="{ width: `${RAIL_WIDTH}px` }"
    :aria-label="$t('cloud-rail.title')"
  >
    <CloudPopover />

    <div class="bg-base-content/10 my-2 h-px w-6"></div>

    <div class="relative flex flex-col gap-1">
      <!-- One highlight that moves between the icons, rather than one per icon. -->
      <span
        v-if="active !== -1"
        class="highlight bg-base-300 absolute top-0 left-0 size-8 rounded-md"
        :style="{ translate: `0 ${active * STEP}px` }"
      ></span>
      <button
        v-for="item in items"
        :key="item.id"
        type="button"
        class="icon-btn btn btn-ghost btn-square btn-sm hover:text-base-content relative hover:bg-transparent"
        :class="panel === item.id ? 'text-base-content' : 'text-base-content/60'"
        :title="item.label"
        :aria-label="item.label"
        :aria-pressed="panel === item.id"
        @click="toggleRail(item.id)"
      >
        <component :is="item.icon" class="size-5" />
        <!-- The same mark the bell and the container rows carry, so one glance
             means the same thing everywhere on the page. -->
        <span v-if="item.dot" class="bg-warning absolute end-1 top-1 size-1.5 rounded-full"></span>
      </button>
    </div>

    <!-- Bottom of the strip, mirroring the nav's collapse on the other edge. -->
    <button
      type="button"
      class="icon-btn btn btn-ghost btn-square btn-sm text-base-content/40 mt-auto"
      :title="$t('cloud-rail.hide')"
      :aria-label="$t('cloud-rail.hide')"
      @click="hideRail()"
    >
      <mdi:chevron-right class="size-5" />
    </button>
  </nav>
</template>

<script lang="ts" setup>
import { RAIL_WIDTH } from "@/composable/cloud/cloudRail";

// A btn-sm square is 2rem, and the column's gap-1 adds a quarter.
const STEP = 36;

const { panel, toggleRail, hideRail } = useCloudRail();
const items = useRailItems();
const active = computed(() => items.value.findIndex((item) => item.id === panel.value));
</script>

<style scoped>
.rail {
  transition:
    background-color 280ms cubic-bezier(0.2, 0.8, 0.2, 1),
    border-color 280ms cubic-bezier(0.2, 0.8, 0.2, 1);
}

.highlight {
  transition: translate 200ms cubic-bezier(0.34, 1.56, 0.64, 1);
}

@media (prefers-reduced-motion: reduce) {
  .rail,
  .highlight {
    transition: none;
  }
}
</style>
