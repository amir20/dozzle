<template>
  <!--
    Chat, metrics and alerts in the app's one drawer, as tabs. The drawer is
    modal, so whatever on it points back at the logs ("show me the lines")
    closes it first, the way the phone sheet always did.

    Everything here is Cloud, which the header says once so no tab has to carry
    a slogan. It opens only where cloud is linked, so the mark is a statement of
    where the answers come from and never an advert.
  -->
  <!-- Bleeds sideways only. Bled past the drawer's bottom padding too, the
       content overflowed the box by that padding, and focusing the composer
       scrolled the header (and the drawer's own close buttons) off the top. -->
  <div class="-mx-6 flex h-full min-h-0 flex-col">
    <header class="shrink-0 px-6">
      <h2 class="flex items-center gap-2 text-2xl font-bold">
        <span class="bg-info/10 text-info flex size-8 shrink-0 items-center justify-center rounded-full">
          <mdi:cloud class="size-4.5" />
        </span>
        {{ $t("cloud.title") }}
      </h2>
    </header>

    <nav
      class="border-base-content/10 mt-4 flex shrink-0 items-center gap-1 border-b px-6 pb-2"
      :aria-label="$t('cloud-rail.title')"
    >
      <button
        v-for="item in items"
        :key="item.id"
        type="button"
        class="flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm transition-colors"
        :class="panel === item.id ? 'bg-base-300 font-semibold' : 'text-base-content/60 hover:bg-base-300'"
        :aria-pressed="panel === item.id"
        @click="openRail(item.id)"
      >
        <component :is="item.icon" class="size-4 shrink-0" />
        <span class="truncate">{{ item.label }}</span>
        <span v-if="item.dot" class="bg-warning size-1.5 shrink-0 rounded-full"></span>
      </button>
      <!-- A thread that has run its course is in the way of the next one. -->
      <button
        v-if="panel === 'chat' && messages.length"
        type="button"
        class="btn btn-ghost btn-xs btn-square ml-auto"
        :title="$t('cloud-chat.clear')"
        :aria-label="$t('cloud-chat.clear')"
        @click="resetChat()"
      >
        <octicon:trash-24 class="size-4" />
      </button>
    </nav>

    <div class="min-h-0 flex-1 overflow-hidden">
      <ChatPane v-if="panel === 'chat'" />
      <RailMetrics v-else-if="panel === 'metrics'" />
      <RailAlerts v-else />
    </div>
  </div>
</template>

<script lang="ts" setup>
import mdiMessageOutline from "~icons/mdi/message-outline";
import mdiChartLine from "~icons/mdi/chart-line";
import mdiBellOutline from "~icons/mdi/bell-outline";

// Handed to every drawer occupant; the drawer's own X and Esc close this one.
defineProps<{ close?: () => void }>();

// Carries markdown-it, and nobody who never asks a question should download it.
const ChatPane = defineAsyncComponent(() => import("@/components/cloud/chat/ChatPane.vue"));

const { panel, openRail, closeRail } = useCloudRail();
const { messages, reset: resetChat } = useCloudChat();
// Scoped to the view, because the panel this bell opens is.
const { unseen: unseenAlerts } = useViewAlerts();
const { t } = useI18n();

const items = computed(() => [
  { id: "chat" as const, label: t("cloud-rail.chat"), icon: mdiMessageOutline },
  { id: "metrics" as const, label: t("cloud-rail.metrics"), icon: mdiChartLine },
  { id: "alerts" as const, label: t("cloud-rail.alerts"), icon: mdiBellOutline, dot: unseenAlerts.value },
]);

// The drawer unmounts its occupant when it closes or another occupant takes
// over, so this is the one place that knows the panel is gone.
onUnmounted(closeRail);
</script>
