<template>
  <!--
    The cloud panels. On a desktop the rail is this drawer's edge: the panel
    slides out from behind it and the rail takes the panel's fill, so the two
    read as one surface, and the rail's icons are the tabs. Not the app's modal
    drawer, because the rail has to stay clickable while a panel is open.

    On a phone there is no rail, so the drawer is a screen of its own and the
    tabs float at its bottom the way the app's own tab bar does.
  -->
  <Transition name="cloud-fade">
    <div v-if="panel && !isMobile" class="bg-base-300/50 fixed inset-0 z-40" @click="closeRail()"></div>
  </Transition>
  <Transition name="cloud-slide">
    <!-- Modal to everything but the rail: the layout makes the page behind it
         inert, and focus comes in on open and goes back on close. -->
    <section
      v-if="panel"
      ref="dialog"
      role="dialog"
      aria-modal="true"
      :aria-label="title"
      tabindex="-1"
      class="bg-base-100 fixed inset-y-0 flex flex-col outline-hidden"
      :class="isMobile ? 'pt-safe inset-x-0 z-50' : 'z-40 shadow-lg'"
      :style="isMobile ? undefined : { right: `${edgeOffset}px`, width: `min(48rem, calc(100vw - ${edgeOffset}px))` }"
    >
      <header class="flex shrink-0 items-center gap-3 px-6 pt-6">
        <!-- A phone has no rail, so its mark lives here. -->
        <CloudMark v-if="isMobile" />
        <!-- The panel's own glyph, as the SQL drawer leads with its file icon. A phone
             already shows the cloud mark here, so the account adds none. -->
        <component
          :is="titleIcon"
          v-if="titleIcon && !(isMobile && panel === 'account')"
          class="text-primary size-6 shrink-0"
          aria-hidden="true"
        />
        <h2 class="min-w-0 flex-1 truncate text-2xl font-bold">{{ title }}</h2>
        <!-- What fired is here; what makes it fire is the notifications page. -->
        <RouterLink
          v-if="panel === 'alerts' && config.enableNotifications"
          :to="{ name: '/notifications' }"
          class="btn btn-sm"
          data-testid="alert-rules"
          @click="closeRail()"
        >
          <mdi:bell-cog-outline class="size-4" />
          {{ $t("cloud-rail.alert-rules") }}
        </RouterLink>
        <!-- A thread that has run its course is in the way of the next one. -->
        <button
          v-if="panel === 'chat' && messages.length"
          type="button"
          class="icon-btn"
          :title="$t('cloud-chat.clear')"
          :aria-label="$t('cloud-chat.clear')"
          @click="resetChat()"
        >
          <octicon:trash-24 class="size-4" />
        </button>
        <button type="button" class="icon-btn" :aria-label="$t('cloud-chat.close')" @click="closeRail()">
          <mdi:close />
        </button>
      </header>

      <!-- The account is the same on every page, so it has no scope to pick. -->
      <!-- px-4 matches the panels' own padding, so the chip lines up with what is under it. -->
      <div v-if="inView.length && panel !== 'account'" class="shrink-0 px-4 pt-4">
        <CloudScopeChip />
      </div>

      <!-- On a phone the tab bar floats over the bottom, so the panel stops above it. -->
      <div
        class="relative mt-2 min-h-0 flex-1"
        :style="isMobile ? { marginBottom: 'var(--mobile-tabbar-height)' } : undefined"
      >
        <Transition name="cloud-fade">
          <div :key="panel" class="absolute inset-0">
            <RailFindings v-if="panel === 'findings'" />
            <RailAlerts v-else-if="panel === 'alerts'" />
            <ChatPane v-else-if="panel === 'chat'" />
            <RailAccount v-else-if="panel === 'account'" />
            <!-- A page about one container gets its history; the instance gets every host's. -->
            <RailMetrics v-else-if="scoped" />
            <RailHostMetrics v-else />
          </div>
        </Transition>
      </div>

      <nav
        v-if="isMobile"
        class="frosted border-base-content/10 absolute right-[calc(0.75rem+env(safe-area-inset-right))] bottom-[calc(0.75rem+env(safe-area-inset-bottom))] left-[calc(0.75rem+env(safe-area-inset-left))] flex rounded-full border p-1 shadow-lg"
        :aria-label="$t('cloud-rail.title')"
      >
        <button
          v-for="item in items"
          :key="item.id"
          type="button"
          class="relative flex min-h-13 min-w-0 flex-1 flex-col items-center justify-center gap-0.5 rounded-full px-1 text-[0.625rem] font-medium transition-colors"
          :class="panel === item.id ? 'bg-base-content/10 text-primary' : 'text-base-content/60'"
          :aria-pressed="panel === item.id"
          @click="openRail(item.id)"
        >
          <component :is="item.icon" class="size-6" />
          <span class="max-w-full truncate">{{ item.label }}</span>
          <span v-if="item.dot" class="bg-warning absolute top-1.5 left-[calc(50%+0.5rem)] size-2 rounded-full"></span>
        </button>
      </nav>
    </section>
  </Transition>
</template>

<script lang="ts" setup>
import mdiCloudOutline from "~icons/mdi/cloud-outline";

// Carries markdown-it, and nobody who never asks a question should download it.
const ChatPane = defineAsyncComponent(() => import("@/components/cloud/chat/ChatPane.vue"));

const { panel, edgeOffset, openRail, closeRail } = useCloudRail();
const { messages, reset: resetChat } = useCloudChat();
const { inView, scoped } = useCloudScope();
// The labels are needed everywhere; the dots only where this draws the tabs.
const items = useRailItems(() => isMobile.value && !!panel.value);
const { t } = useI18n();

const titleIcon = computed(() =>
  panel.value === "account" ? mdiCloudOutline : items.value.find((i) => i.id === panel.value)?.icon,
);
const title = computed(() =>
  panel.value === "account" ? t("cloud.title") : (items.value.find((i) => i.id === panel.value)?.label ?? ""),
);

// Not a native dialog, so Esc is ours to handle. A modal on top (the palette,
// the app drawer) owns its own Esc, and so does an open menu (a native popover,
// which Esc dismisses first): the first Esc closes the menu, the next the panel.
function layerOnTop() {
  if (document.querySelector("dialog[open]")) return true;
  try {
    return document.querySelector(":popover-open") !== null;
  } catch {
    // An engine without the selector has no open popovers to ask about.
    return false;
  }
}
onKeyStroke("Escape", () => {
  if (panel.value && !layerOnTop()) closeRail();
});

// Not a native dialog either, so focus is ours too. On open it moves to the
// panel itself, which is enough for a screen reader to land in it and for Tab
// to start there; chat then moves it on to its composer when it mounts. On
// close it goes back to whatever opened the panel, if that is still on the page.
const dialog = useTemplateRef<HTMLElement>("dialog");
let opener: HTMLElement | null = null;
watch(panel, async (next, prev) => {
  if (next && !prev) {
    const active = document.activeElement;
    opener = active instanceof HTMLElement && active !== document.body ? active : null;
    await nextTick();
    // Chat may already have focused its composer; keep that.
    if (!dialog.value?.contains(document.activeElement)) dialog.value?.focus({ preventScroll: true });
  } else if (!next && prev) {
    const target = opener;
    const leaving = dialog.value;
    opener = null;
    // After the page stops being inert, or the focus call is a no-op.
    await nextTick();
    // Only when focus went down with the panel: a reader who has since put it
    // somewhere else on purpose keeps it there.
    const active = document.activeElement;
    const stranded = !active || active === document.body || !!leaving?.contains(active);
    if (stranded && target?.isConnected) target.focus({ preventScroll: true });
  }
});
</script>

<style scoped>
.cloud-slide-enter-active,
.cloud-slide-leave-active {
  transition: translate 280ms cubic-bezier(0.2, 0.8, 0.2, 1);
}

.cloud-slide-enter-from,
.cloud-slide-leave-to {
  translate: 100% 0;
}

.cloud-fade-enter-active,
.cloud-fade-leave-active {
  transition: opacity 160ms ease;
}

.cloud-fade-enter-from,
.cloud-fade-leave-to {
  opacity: 0;
}

@media (prefers-reduced-motion: reduce) {
  .cloud-slide-enter-active,
  .cloud-slide-leave-active,
  .cloud-fade-enter-active,
  .cloud-fade-leave-active {
    transition: none;
  }
}
</style>
