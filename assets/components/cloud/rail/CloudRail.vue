<template>
  <!--
    A strip of icons on the right edge, and the panel one of them opens.

    Fixed rather than a splitpanes column: a column reads as a second log view,
    which is what the pinned containers are, and these are not that. The layout
    pads the page by exactly this width, so the rail and its panel never sit
    over the stream and nothing here overlaps what it is about.

    Everything on it is Cloud, which the mark at the top says once so no panel
    has to carry a slogan. It is mounted only where cloud is linked, so the mark
    is a statement of where the answers come from and never an advert.

    A phone has no room for a permanent strip, and a 550px column beside a 390px
    screen is not a column. There the strip is gone entirely and the panel is the
    screen, opened from the toolbar or the palette and closed with the same X. A
    vertical strip of icons down the edge of a 390px screen is a desktop affordance
    wearing a phone's clothes, so on a sheet the three panels are tabs across the
    top instead, where a thumb can reach them.
  -->
  <div class="fixed z-30 flex" :class="sheet ? 'pt-safe pb-safe bg-base-100 inset-0 z-40' : 'inset-y-0 right-0'">
    <section
      v-if="panel"
      class="border-base-content/10 bg-base-100 flex min-w-0 flex-1 flex-col"
      :class="{ 'border-l': !sheet }"
      :style="sheet ? undefined : { width: `${panelWidth}px`, flex: 'none' }"
    >
      <header class="border-base-content/10 flex shrink-0 items-center gap-2 border-b px-4 py-3">
        <!-- On a sheet the tabs under this name the panel, so the header says the
             one thing they cannot: where the answers come from. -->
        <template v-if="sheet">
          <span class="bg-info/10 text-info flex size-6 shrink-0 items-center justify-center rounded-full">
            <mdi:cloud class="size-3.5" />
          </span>
          <span class="text-sm font-semibold">{{ $t("cloud.title") }}</span>
        </template>
        <template v-else>
          <component :is="active.icon" class="text-base-content/60 size-4 shrink-0" />
          <span class="text-sm font-semibold">{{ active.label }}</span>
          <!-- Who answered. Muted, because the reader needs it once to trust the
               panel and never again while reading it. -->
          <span class="text-base-content/40 ml-1 flex items-center gap-1 text-xs">
            <mdi:cloud class="text-info/70 size-3.5" />
            {{ $t("cloud.title") }}
          </span>
        </template>
        <div class="ml-auto flex shrink-0 items-center gap-1">
          <!-- A thread that has run its course is in the way of the next one,
               and the only way out of it used to be a reload. -->
          <button
            v-if="panel === 'chat' && messages.length"
            type="button"
            class="btn btn-ghost btn-xs btn-square"
            :title="$t('cloud-chat.clear')"
            :aria-label="$t('cloud-chat.clear')"
            @click="resetChat()"
          >
            <octicon:trash-24 class="size-4" />
          </button>
          <button
            type="button"
            class="btn btn-ghost btn-xs btn-square"
            :aria-label="$t('cloud-rail.close')"
            @click="closeRail()"
          >
            <mdi:close class="size-4" />
          </button>
        </div>
      </header>

      <!-- The strip's job, laid out for a thumb: same three panels, same dot, and
           the only way to move between them once the strip is gone. -->
      <nav
        v-if="sheet"
        class="border-base-content/10 flex shrink-0 gap-1 border-b p-2"
        :aria-label="$t('cloud-rail.title')"
      >
        <button
          v-for="item in items"
          :key="item.id"
          type="button"
          class="flex flex-1 items-center justify-center gap-1.5 rounded-md p-2 text-sm transition-colors"
          :class="panel === item.id ? 'bg-info/10 text-info font-semibold' : 'text-base-content/60'"
          :aria-pressed="panel === item.id"
          @click="toggleRail(item.id)"
        >
          <component :is="item.icon" class="size-4 shrink-0" />
          <span class="truncate">{{ item.label }}</span>
          <span v-if="item.dot" class="bg-warning size-1.5 shrink-0 rounded-full"></span>
        </button>
      </nav>

      <div class="min-h-0 flex-1 overflow-hidden">
        <ChatPane v-if="panel === 'chat'" />
        <RailMetrics v-else-if="panel === 'metrics'" />
        <RailAlerts v-else />
      </div>
    </section>

    <nav
      v-if="!sheet"
      class="border-base-content/10 bg-base-200/40 flex shrink-0 flex-col items-center gap-1 border-l py-3"
      :style="{ width: `${RAIL_WIDTH}px` }"
      :aria-label="$t('cloud-rail.title')"
    >
      <router-link
        :to="{ name: '/settings', hash: '#cloud' }"
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

      <!-- Bottom of the strip, mirroring the nav's collapse on the other edge.
           Hiding is remembered, and the tab on the edge brings it back. -->
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
  </div>
</template>

<script lang="ts" setup>
import { RAIL_WIDTH } from "@/composable/cloud/cloudRail";
import mdiMessageOutline from "~icons/mdi/message-outline";
import mdiChartLine from "~icons/mdi/chart-line";
import mdiBellOutline from "~icons/mdi/bell-outline";

// Carries markdown-it, and nobody who never asks a question should download it.
const ChatPane = defineAsyncComponent(() => import("@/components/cloud/chat/ChatPane.vue"));

const { panel, panelWidth, sheet, closeRail, toggleRail, hideRail } = useCloudRail();
const { messages, reset: resetChat } = useCloudChat();
// Scoped to the view, because the panel this bell opens is. The nav's bell keeps
// the instance-wide one: it opens the notifications page, which shows everything.
const { unseen: unseenAlerts } = useViewAlerts();
const { t } = useI18n();

const items = computed(() => [
  { id: "chat" as const, label: t("cloud-rail.chat"), icon: mdiMessageOutline },
  { id: "metrics" as const, label: t("cloud-rail.metrics"), icon: mdiChartLine },
  { id: "alerts" as const, label: t("cloud-rail.alerts"), icon: mdiBellOutline, dot: unseenAlerts.value },
]);

const active = computed(() => items.value.find((i) => i.id === panel.value) ?? items.value[0]);

// Escape closes whatever is open, the way every other dismissable surface here
// behaves. The panels are not modal, so nothing else is listening for it.
onKeyStroke("Escape", () => {
  if (panel.value) closeRail();
});
</script>
