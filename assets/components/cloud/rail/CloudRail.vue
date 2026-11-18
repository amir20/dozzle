<template>
  <!--
    A strip of icons on the right edge: how people find Cloud at all. Each icon
    opens its panel in the drawer, and the icon of the panel on show stays
    pressed.

    Fixed rather than a splitpanes column, and the layout pads the page by
    exactly its width, so the strip never sits over the stream. It is mounted
    only where cloud is linked, so the mark at the top is a statement of where
    the answers come from and never an advert. Hiding it is remembered, and the
    tab on the edge brings it back.
  -->
  <nav
    class="border-base-content/10 bg-base-200/40 fixed inset-y-0 right-0 z-30 flex flex-col items-center gap-1 border-l py-3"
    :style="{ width: `${RAIL_WIDTH}px` }"
    :aria-label="$t('cloud-rail.title')"
  >
    <router-link
      :to="{ name: '/settings/cloud' }"
      class="bg-info/10 text-info rounded-full p-1.5 transition-opacity hover:opacity-80"
      :title="$t('cloud.title')"
      :aria-label="$t('cloud.title')"
    >
      <mdi:cloud class="size-5" />
    </router-link>

    <div class="bg-base-content/10 my-2 h-px w-6"></div>

    <button
      v-for="item in items"
      :key="item.id"
      type="button"
      class="icon-btn btn btn-ghost btn-square btn-sm relative"
      :class="panel === item.id ? 'bg-base-300 text-base-content' : 'text-base-content/60'"
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
import mdiMessageOutline from "~icons/mdi/message-outline";
import mdiChartLine from "~icons/mdi/chart-line";
import mdiBellOutline from "~icons/mdi/bell-outline";

const { panel, toggleRail, hideRail } = useCloudRail();
// Scoped to the view, because the panel this bell opens is. The nav's bell keeps
// the instance-wide one: it opens the notifications page, which shows everything.
const { unseen: unseenAlerts, scoped: containersInView } = useViewAlerts();
const { t } = useI18n();

const items = computed(() => [
  { id: "chat" as const, label: t("cloud-rail.chat"), icon: mdiMessageOutline },
  { id: "metrics" as const, label: t("cloud-rail.metrics"), icon: mdiChartLine },
  // Off a log view the notifications page already lists every alert, so the
  // tab would only repeat it.
  ...(containersInView.value
    ? [{ id: "alerts" as const, label: t("cloud-rail.alerts"), icon: mdiBellOutline, dot: unseenAlerts.value }]
    : []),
]);
</script>
