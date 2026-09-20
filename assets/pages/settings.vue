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

    <!-- On a phone the sub-nav folds into a row of chips above the first section. -->
    <nav role="tablist" class="tabs tabs-box tabs-sm flex-nowrap overflow-x-auto @3xl:hidden">
      <router-link
        v-for="item in sections"
        :key="item.id"
        :to="{ hash: `#${item.id}` }"
        replace
        @click="pin(item.id)"
        role="tab"
        class="tab shrink-0 whitespace-nowrap"
        :class="{ 'tab-active': active === item.id }"
        :aria-current="active === item.id ? 'location' : false"
      >
        {{ item.label }}
      </router-link>
    </nav>

    <div class="flex items-start gap-10">
      <!-- What this install is, then the preferences, then what is about the server
           itself, each group after a hairline. -->
      <ul class="menu sticky top-4 hidden w-48 shrink-0 p-0 @3xl:flex">
        <template v-for="item in sections" :key="item.id">
          <li v-if="item.divider"></li>
          <li>
            <router-link
              :to="{ hash: `#${item.id}` }"
              replace
              @click="pin(item.id)"
              :class="{ 'menu-active': active === item.id }"
              :aria-current="active === item.id ? 'location' : false"
            >
              <component :is="item.icon" class="size-4 opacity-60" />
              <span class="flex-1">{{ item.label }}</span>
              <span v-if="item.id === 'about' && hasRelease" class="status status-warning"></span>
            </router-link>
          </li>
        </template>
      </ul>

      <div class="flex max-w-4xl min-w-0 flex-1 flex-col gap-8">
        <!-- ABOUT leads: what this install is, whether it is stale and what it will do
             about that. Everything below it is a setting someone changes. -->
        <section id="about" ref="aboutEl" class="border-base-content/10 flex scroll-mt-4 flex-col gap-4 border-b pb-6">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="flex flex-wrap items-center gap-2.5">
              <span class="text-[0.9375rem] font-semibold">Dozzle</span>
              <span class="badge badge-soft badge-sm">{{ config.version }}</span>
              <!-- With auto-update on, the panel below already says what happens next,
                   so the badge would be a second, louder answer to the same question. -->
              <a
                v-if="hasRelease && !autoUpdate"
                :href="latestRelease?.htmlUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="badge badge-soft badge-warning badge-sm hover:bg-warning/15"
              >
                <span class="status status-warning"></span>
                {{ $t("settings.new-version", { version: latestRelease?.name }) }}
              </a>
            </div>
            <div class="flex gap-2">
              <a href="https://github.com/amir20/dozzle" target="_blank" rel="noopener noreferrer" class="btn btn-sm">
                <mdi:github /> GitHub
              </a>
              <a href="https://github.com/sponsors/amir20" target="_blank" rel="noopener noreferrer" class="btn btn-sm">
                <mdi:heart class="text-error" /> {{ $t("settings.sponsor") }}
              </a>
            </div>
          </div>
          <SelfUpdateStatus v-if="setupStatus && autoUpdate" :status="setupStatus" :auto-update="autoUpdate" />
        </section>

        <!-- APPEARANCE: app-wide only. Anything that changes how a log line looks is in Logs,
             under the preview it changes. -->
        <section id="appearance" ref="appearanceEl" class="scroll-mt-4">
          <h2 class="section-heading">{{ $t("settings.appearance") }}</h2>
          <div class="card card-border bg-base-200/40 divide-base-content/10 divide-y overflow-hidden">
            <SettingRow :label="$t('settings.color-scheme')" class="px-4">
              <div class="flex gap-3">
                <button
                  v-for="opt in themes"
                  :key="opt.value"
                  type="button"
                  class="flex flex-col items-center gap-1.5"
                  :aria-pressed="lightTheme === opt.value"
                  @click="lightTheme = opt.value"
                >
                  <span
                    class="border-base-content/15 flex h-12 w-20 overflow-hidden rounded-md border transition-shadow"
                    :class="{ 'ring-primary ring-offset-base-100 ring-2 ring-offset-2': lightTheme === opt.value }"
                  >
                    <span
                      v-for="swatch in opt.swatches"
                      :key="swatch"
                      :data-theme="swatch"
                      class="bg-base-100 flex flex-1 flex-col gap-1 p-2"
                    >
                      <span class="bg-base-content/20 h-1 w-3/4 rounded-full"></span>
                      <span class="bg-primary h-1 w-1/2 rounded-full"></span>
                      <span class="bg-base-content/20 h-1 w-full rounded-full"></span>
                    </span>
                  </span>
                  <span
                    class="text-xs"
                    :class="lightTheme === opt.value ? 'font-medium' : 'text-base-content/60 font-normal'"
                  >
                    {{ opt.label }}
                  </span>
                </button>
              </div>
            </SettingRow>
            <SettingRow :label="$t('settings.locale')" class="px-4">
              <DropdownMenu
                plain
                v-model="locale"
                :options="[
                  { label: 'Auto', value: '' },
                  ...availableLocales.map((l) => ({ label: localeName(l), value: l })),
                ]"
              />
            </SettingRow>
          </div>
        </section>

        <!-- LOGS -->
        <section id="logs" ref="logsEl" class="scroll-mt-4">
          <h2 class="section-heading">{{ $t("settings.logs") }}</h2>
          <div class="card card-border bg-base-200/40 divide-base-content/10 divide-y overflow-hidden">
            <div ref="previewEl">
              <LogList
                :messages="fakeMessages"
                :last-selected-item="undefined"
                :show-container-name="false"
                class="bg-base-100 pb-2"
              />
            </div>
            <SettingRow :label="$t('settings.font-size')" class="px-4">
              <span role="tablist" class="tabs tabs-box tabs-sm">
                <button
                  v-for="opt in sizes"
                  :key="opt.value"
                  type="button"
                  role="tab"
                  class="tab"
                  :class="{ 'tab-active': size === opt.value }"
                  @click="size = opt.value"
                >
                  {{ opt.label }}
                </button>
              </span>
            </SettingRow>
            <SettingRow
              tag="label"
              :label="$t('settings.compact')"
              :description="$t('settings.compact-desc')"
              class="px-4"
            >
              <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="compact" />
            </SettingRow>
            <SettingRow tag="label" :label="$t('settings.show-timestamps')" class="px-4">
              <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="showTimestamp" />
            </SettingRow>
            <SettingRow tag="label" :label="$t('settings.soft-wrap')" class="px-4">
              <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="softWrap" />
            </SettingRow>
            <SettingRow
              tag="label"
              :label="$t('settings.highlight-errors')"
              :description="$t('settings.highlight-errors-desc')"
              class="px-4"
            >
              <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="highlightErrors" />
            </SettingRow>
            <SettingRow :label="$t('settings.datetime-format')" class="px-4">
              <DropdownMenu
                plain
                v-model="dateLocale"
                :options="[
                  { label: 'Auto', value: 'auto' },
                  { label: 'MM/DD/YYYY', value: 'en-US' },
                  { label: 'DD/MM/YYYY', value: 'en-GB' },
                  { label: 'DD.MM.YYYY', value: 'de-DE' },
                  { label: 'YYYY-MM-DD', value: 'en-CA' },
                ]"
              />
              <DropdownMenu
                plain
                v-model="hourStyle"
                :options="[
                  { label: $t('settings.hour.auto'), value: 'auto' },
                  { label: $t('settings.hour.12'), value: '12' },
                  { label: $t('settings.hour.24'), value: '24' },
                ]"
              />
            </SettingRow>
          </div>
        </section>

        <!-- SIDEBAR -->
        <section id="sidebar" ref="sidebarEl" class="scroll-mt-4">
          <h2 class="section-heading">{{ $t("settings.sidebar") }}</h2>
          <div class="card card-border bg-base-200/40 divide-base-content/10 divide-y overflow-hidden">
            <SettingRow tag="label" :label="$t('settings.show-stopped-containers')" class="px-4">
              <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="showAllContainers" />
            </SettingRow>
            <SettingRow
              :label="$t('settings.group-containers')"
              :description="$t('settings.group-containers-desc')"
              class="px-4"
            >
              <DropdownMenu
                plain
                v-model="groupContainers"
                :options="[
                  { label: $t('settings.grouping.always'), value: 'always' },
                  { label: $t('settings.grouping.at-least-2'), value: 'at-least-2' },
                  { label: $t('settings.grouping.never'), value: 'never' },
                ]"
              />
            </SettingRow>
            <SettingRow
              tag="label"
              :label="$t('settings.show-app-icons')"
              :description="$t('settings.show-app-icons-desc')"
              class="px-4"
            >
              <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="showAppIcons" />
            </SettingRow>
          </div>
        </section>

        <!-- BEHAVIOR -->
        <section id="behavior" ref="behaviorEl" class="scroll-mt-4">
          <h2 class="section-heading">{{ $t("settings.behavior") }}</h2>
          <div class="card card-border bg-base-200/40 divide-base-content/10 divide-y overflow-hidden">
            <SettingRow
              :label="$t('settings.automatic-redirect')"
              :description="$t('settings.automatic-redirect-desc')"
              class="px-4"
            >
              <DropdownMenu
                plain
                v-model="automaticRedirect"
                :options="[
                  { label: $t('settings.redirect.instant'), value: 'instant' },
                  { label: $t('settings.redirect.delayed'), value: 'delayed' },
                  { label: $t('settings.redirect.none'), value: 'none' },
                ]"
              />
            </SettingRow>
            <SettingRow
              tag="label"
              :label="$t('settings.search')"
              :description="$t('settings.search-desc')"
              class="px-4"
            >
              <template #label-suffix><key-shortcut char="f" /></template>
              <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="search" />
            </SettingRow>
            <SettingRow
              v-if="config.imageCheckMode !== 'off'"
              tag="label"
              :label="$t('settings.show-image-update-alert')"
              :description="$t('settings.show-image-update-alert-desc')"
              class="px-4"
            >
              <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="showImageUpdateAlert" />
            </SettingRow>
          </div>
        </section>

        <!-- ADVANCED: the two settings almost nobody changes, folded away. -->
        <details
          class="collapse-arrow card-border bg-base-200/40 divide-base-content/10 group/advanced collapse divide-y"
        >
          <summary class="collapse-title text-base-content/70 flex items-center gap-2 text-sm font-medium">
            <span class="flex-1">{{ $t("settings.advanced") }}</span>
            <span class="text-base-content/40 hidden text-xs font-normal group-open/advanced:hidden @xl:inline">
              {{ $t("settings.show-std") }}, {{ $t("settings.small-scrollbars") }}
            </span>
          </summary>
          <div class="collapse-content divide-base-content/10 divide-y p-0">
            <SettingRow
              tag="label"
              :label="$t('settings.show-std')"
              :description="$t('settings.show-std-desc')"
              class="px-4"
            >
              <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="showStd" />
            </SettingRow>
            <SettingRow
              tag="label"
              :label="$t('settings.small-scrollbars')"
              :description="$t('settings.small-scrollbars-desc')"
              class="px-4"
            >
              <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="smallerScrollbars" />
            </SettingRow>
          </div>
        </details>

        <!-- SETUP and CLOUD act on the server, not on this browser, so they sit below
             the preferences with their own headings rather than under one vague one. -->
        <section v-if="showSetup" id="setup" ref="setupEl" class="flex scroll-mt-4 flex-col gap-4">
          <h2 class="section-heading mb-0!">{{ $t("settings.setup") }}</h2>
          <SetupSettingsCard :status="setupStatus" />
        </section>

        <section v-if="showCloud" id="cloud" ref="cloudEl" class="flex scroll-mt-4 flex-col gap-4">
          <div>
            <h2 class="section-heading mb-1!">{{ $t("cloud.title") }}</h2>
            <p class="text-base-content/60 text-xs">{{ $t("settings.cloud-desc") }}</p>
          </div>
          <CloudSettingsCard />
        </section>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ComplexLogEntry, SimpleLogEntry, GroupedLogEntry } from "@/models/LogEntry";
import IconPalette from "~icons/mdi/palette-outline";
import IconLogs from "~icons/mdi/format-list-text";
import IconSidebar from "~icons/mdi/dock-left";
import IconBehavior from "~icons/mdi/lightning-bolt-outline";
import IconSetup from "~icons/mdi/rocket-launch-outline";
import IconCloud from "~icons/mdi/cloud-outline";
import IconAbout from "~icons/mdi/information-outline";

import {
  settings,
  DEFAULT_SETTINGS,
  type Settings,
  automaticRedirect,
  compact,
  hourStyle,
  dateLocale,
  lightTheme,
  search,
  showAllContainers,
  showStd,
  showTimestamp,
  size,
  smallerScrollbars,
  softWrap,
  locale,
  groupContainers,
  showImageUpdateAlert,
  showAppIcons,
  highlightErrors,
} from "@/stores/settings";

import { availableLocales, i18n } from "@/modules/i18n";

// Each language named in itself, e.g. "Deutsch", "日本語".
function localeName(l: string) {
  const name = new Intl.DisplayNames([l], { type: "language" }).of(l) ?? l;
  return name.charAt(0).toLocaleUpperCase(l) + name.slice(1);
}

const { t } = useI18n();

setTitle(t("title.settings"));
const { latestRelease, hasRelease } = useAnnouncements();
// The setup card reports what is already configured and About shows the update
// schedule, so this page reads the status itself instead of waiting for someone
// to open the wizard. The API only exists in server mode.
const { status: setupStatus, fetchStatus } = useSetup();
const showSetup = computed(() => config.mode === "server");
if (showSetup.value && !setupStatus.value) fetchStatus();

const showCloud = computed(() => config.enableCloud && config.canLinkCloud);

// Auto-update belongs in About: someone reading the version there is asking exactly
// the question the schedule answers.
const autoUpdate = computed(() => {
  const update = setupStatus.value?.autoUpdate;
  return update && update.mode !== "off" ? update : undefined;
});

const themes = computed(() => [
  { label: t("settings.theme.light"), value: "light" as const, swatches: ["light"] },
  { label: t("settings.theme.dark"), value: "dark" as const, swatches: ["dark"] },
  { label: t("settings.theme.auto"), value: "auto" as const, swatches: ["light", "dark"] },
]);

const sizes = computed(() => [
  { label: t("settings.size.small"), value: "small" as const },
  { label: t("settings.size.medium"), value: "medium" as const },
  { label: t("settings.size.large"), value: "large" as const },
]);

const sections = computed(() => {
  const items = [
    { id: "about", label: t("settings.about"), icon: IconAbout },
    { id: "appearance", label: t("settings.appearance"), icon: IconPalette },
    { id: "logs", label: t("settings.logs"), icon: IconLogs },
    { id: "sidebar", label: t("settings.sidebar"), icon: IconSidebar },
    { id: "behavior", label: t("settings.behavior"), icon: IconBehavior },
    showSetup.value ? { id: "setup", label: t("settings.setup"), icon: IconSetup } : undefined,
    showCloud.value ? { id: "cloud", label: t("cloud.title"), icon: IconCloud } : undefined,
  ].filter((s) => s !== undefined);
  // A hairline opening each group: the preferences, then what is about the server.
  const server = items.find((s) => s.id === "setup" || s.id === "cloud");
  return items.map((s) => ({ ...s, divider: s.id === "appearance" || s === server }));
});

// The preview sits above the controls that resize it, so every change would push
// the control out from under the pointer. Scroll by however much the preview's
// bottom edge moved, which keeps everything below it still.
const previewEl = useTemplateRef("previewEl");
let previewBottom: number | undefined;
const previewSettings = () => [size.value, compact.value, showTimestamp.value, softWrap.value];
watch(previewSettings, () => (previewBottom = previewEl.value?.getBoundingClientRect().bottom), { flush: "pre" });
watch(
  previewSettings,
  () => {
    if (previewBottom === undefined || !previewEl.value) return;
    window.scrollBy(0, previewEl.value.getBoundingClientRect().bottom - previewBottom);
    previewBottom = undefined;
  },
  { flush: "post" },
);

// Scroll spy: the active section is the last one whose top has scrolled past a
// small offset. Sections near the bottom may never reach that offset, so a click
// pins its own item until the scroll it started has finished.
const appearanceEl = useTemplateRef("appearanceEl");
const logsEl = useTemplateRef("logsEl");
const sidebarEl = useTemplateRef("sidebarEl");
const behaviorEl = useTemplateRef("behaviorEl");
const setupEl = useTemplateRef("setupEl");
const cloudEl = useTemplateRef("cloudEl");
const aboutEl = useTemplateRef("aboutEl");
const active = ref("about");

const updateActive = () => {
  const els = [aboutEl, appearanceEl, logsEl, sidebarEl, behaviorEl, setupEl, cloudEl]
    .map((r) => r.value)
    .filter((el): el is HTMLElement => el != null);
  if (els.length === 0) return;
  let current = els[0];
  for (const el of els) {
    if (el.getBoundingClientRect().top <= 120) current = el;
  }
  active.value = current.id;
};

let pinned = false;
const pin = (id: string) => {
  active.value = id;
  pinned = true;
};
const unpin = useDebounceFn(() => (pinned = false), 150);

useEventListener(
  window,
  "scroll",
  () => {
    if (pinned) unpin();
    else updateActive();
  },
  { passive: true },
);
onMounted(updateActive);

// Only the preferences this page shows. Nav width and collapsed panels are layout
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

const now = new Date();
const hoursAgo = (hours: number) => {
  const date = new Date(now);
  date.setHours(date.getHours() - hours);
  return date;
};

const fakeMessages = computedWithControl(
  () => i18n.global.locale.value,
  () => [
    new SimpleLogEntry(t("settings.log.preview"), "123", 1, hoursAgo(16), "info", "stdout", ""),
    new SimpleLogEntry(t("settings.log.warning"), "123", 2, hoursAgo(12), "warn", "stdout", ""),
    new GroupedLogEntry(
      [
        t("settings.log.multi-line-error.start-line"),
        t("settings.log.multi-line-error.middle-line"),
        t("settings.log.multi-line-error.end-line"),
      ],
      "123",
      3,
      hoursAgo(7),
      "error",
      "stderr",
    ),
    new ComplexLogEntry(
      {
        message: t("settings.log.complex"),
        context: {
          key: "value",
          key2: "value2",
        },
      },
      "123",
      6,
      new Date(),
      "info",
      "stdout",
      "",
    ),
    new SimpleLogEntry(t("settings.log.simple"), "123", 7, new Date(), "debug", "stderr", ""),
  ],
);
</script>

<style scoped>
@reference "@/main.css";

.section-heading {
  @apply text-base-content/60 mb-2 text-xs font-semibold tracking-wide uppercase;
}
</style>
