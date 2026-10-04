<template>
  <div class="@container flex flex-col gap-6 p-4 md:px-8">
    <section>
      <Links />
    </section>

    <div class="flex flex-wrap items-end justify-between gap-x-4 gap-y-1">
      <div>
        <h1 class="text-2xl font-bold">{{ $t("title.settings") }}</h1>
        <p class="text-base-content/60 text-sm">{{ $t("settings.subtitle") }}</p>
      </div>
      <button type="button" class="btn btn-ghost btn-xs text-base-content/60" @click="reset">
        {{ $t("settings.reset") }}
      </button>
    </div>

    <!-- On a phone the nav folds into a row of chips above the page. -->
    <nav
      ref="chipsEl"
      role="tablist"
      class="tabs tabs-box tabs-sm flex-nowrap overflow-x-auto @3xl:hidden"
      :aria-label="$t('title.settings')"
    >
      <router-link
        v-for="item in pages"
        :key="item.id"
        :to="item.to"
        role="tab"
        class="tab shrink-0 whitespace-nowrap"
        active-class="tab-active"
      >
        {{ item.label }}
      </router-link>
    </nav>

    <div class="flex items-start gap-10">
      <!-- The preferences for this browser, then what is about the server, then what this
           install is, each group after a hairline. -->
      <ul class="menu sticky top-4 hidden w-48 shrink-0 p-0 @3xl:flex" :aria-label="$t('title.settings')">
        <template v-for="item in pages" :key="item.id">
          <li v-if="item.divider"></li>
          <li>
            <router-link :to="item.to" active-class="menu-active">
              <component :is="item.icon" class="size-4 opacity-60" />
              <span class="flex-1">{{ item.label }}</span>
              <span v-if="item.id === 'about' && hasRelease" class="status status-warning"></span>
            </router-link>
          </li>
        </template>
      </ul>

      <div class="flex max-w-4xl min-w-0 flex-1 flex-col gap-8">
        <RouterView />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { Component } from "vue";
import IconGeneral from "~icons/mdi/tune-variant";
import IconLogs from "~icons/mdi/format-list-text";
import IconSidebar from "~icons/mdi/dock-left";
import IconUpdates from "~icons/mdi/package-down";
import IconCloud from "~icons/mdi/cloud-outline";
import IconSetup from "~icons/mdi/rocket-launch-outline";
import IconAbout from "~icons/mdi/information-outline";

import { settings, DEFAULT_SETTINGS, type Settings } from "@/stores/settings";
import { settingsPages, type SettingsPageId } from "@/composable/app/settingsPages";

const { t } = useI18n();

setTitle(t("title.settings"));
const { hasRelease } = useAnnouncements();
// Setup, Updates and About all read the setup status, so the layout fetches it once
// instead of each page waiting for someone to open the wizard. The API only exists
// in server mode.
const { status: setupStatus, fetchStatus } = useSetup();
if (config.mode === "server" && !setupStatus.value) fetchStatus();

const icons: Record<SettingsPageId, Component> = {
  general: IconGeneral,
  logs: IconLogs,
  sidebar: IconSidebar,
  updates: IconUpdates,
  cloud: IconCloud,
  setup: IconSetup,
  about: IconAbout,
};

const labels = computed<Record<SettingsPageId, string>>(() => ({
  general: t("settings.general"),
  logs: t("settings.logs"),
  sidebar: t("settings.sidebar"),
  updates: t("settings.updates"),
  cloud: t("cloud.title"),
  setup: t("settings.setup"),
  about: t("settings.about"),
}));

const visible = settingsPages(config);
const pages = computed(() => {
  // A hairline opening each group: the server pages, then About.
  const server = visible.find((id) => id === "updates" || id === "cloud" || id === "setup");
  return visible.map((id) => ({
    id,
    to: `/settings/${id}` as const,
    label: labels.value[id],
    icon: icons[id],
    divider: id === server || id === "about",
  }));
});

// The chip row scrolls sideways, so a page opened from a link (About, Setup) can sit
// past the edge. Bring the active chip into view whenever the page changes.
const route = useRoute();
const chipsEl = useTemplateRef("chipsEl");
const revealActiveChip = () =>
  chipsEl.value?.querySelector(".tab-active")?.scrollIntoView?.({ block: "nearest", inline: "nearest" });
watch(() => route.path, revealActiveChip, { flush: "post" });
onMounted(revealActiveChip);

// Only the preferences these pages show. Nav width and collapsed panels are layout
// state the user set somewhere else and would not expect a reset here to touch.
const resettable: (keyof Settings)[] = [
  "lightTheme",
  "locale",
  "size",
  "compact",
  "showTimestamp",
  "softWrap",
  "highlightErrors",
  "dateLocale",
  "hourStyle",
  "showAllContainers",
  "groupContainers",
  "showAppIcons",
  "automaticRedirect",
  "search",
  "showImageUpdateAlert",
  "showStd",
  "smallerScrollbars",
];

const reset = () => {
  Object.assign(settings.value, Object.fromEntries(resettable.map((key) => [key, DEFAULT_SETTINGS[key]])));
};
</script>
