<template>
  <div class="@container flex flex-col gap-5 px-4 py-4 md:px-8">
    <section>
      <Links>
        <template #more-items>
          <Tag class="font-mono">{{ config.version }}</Tag>
        </template>
      </Links>
    </section>

    <!-- ABOUT -->
    <section class="flex flex-col gap-4">
      <div>
        <h2 class="text-xl font-semibold tracking-tight">{{ $t("settings.about") }}</h2>
        <p class="text-base-content/60 mt-1 text-sm">{{ $t("settings.about-desc") }}</p>
      </div>

      <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
        <div class="flex flex-col gap-2 p-5">
          <div class="flex flex-wrap items-center gap-3">
            <span class="text-2xl font-semibold tracking-tight">Dozzle</span>
            <span class="status-pill status-pill-neutral">{{ config.version }}</span>
            <a
              v-if="hasRelease"
              :href="latestRelease?.htmlUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="status-pill status-pill-warning hover:bg-warning/15"
            >
              <span class="size-1.5 rounded-full bg-current"></span>
              {{ latestRelease?.name }} available
            </a>
          </div>
          <div class="text-base-content/60 font-mono text-xs">
            <template v-if="hasRelease && latestRelease?.createdAt">
              Latest release {{ latestRelease.name }} ·
              {{ new Date(latestRelease.createdAt).toLocaleDateString(undefined, dateFmt) }}
            </template>
            <template v-else> You're running the latest version. </template>
          </div>
        </div>

        <div class="flex flex-col gap-3 p-4">
          <div>
            <div class="text-sm font-medium">{{ $t("settings.support-title") }}</div>
            <div class="text-base-content/60 text-xs">{{ $t("settings.help-support") }}</div>
          </div>
          <div class="flex flex-wrap gap-2">
            <a href="https://github.com/amir20/dozzle" target="_blank" rel="noopener noreferrer" class="btn btn-sm">
              <mdi:github /> amir20/dozzle
            </a>
            <a
              href="https://github.com/sponsors/amir20"
              target="_blank"
              rel="noopener noreferrer"
              class="btn btn-primary btn-sm"
            >
              <mdi:heart /> Sponsor on GitHub
            </a>
            <a
              href="https://buymeacoffee.com/amirraminfar"
              target="_blank"
              rel="noopener noreferrer"
              class="btn btn-secondary btn-sm"
            >
              <mdi:beer /> Buy me a beer
            </a>
          </div>
        </div>
      </div>
    </section>

    <!-- CLOUD -->
    <section class="flex flex-col gap-4">
      <div>
        <h2 class="text-xl font-semibold tracking-tight">{{ $t("cloud.title") }}</h2>
        <p class="text-base-content/60 mt-1 text-sm">{{ $t("settings.cloud-desc") }}</p>
      </div>
      <CloudSettingsCard />
    </section>

    <!-- DISPLAY -->
    <section class="flex flex-col gap-4">
      <div>
        <h2 class="text-xl font-semibold tracking-tight">{{ $t("settings.display") }}</h2>
        <p class="text-base-content/60 mt-1 text-sm">{{ $t("settings.display-desc") }}</p>
      </div>

      <div class="grid items-stretch gap-3 @3xl:grid-cols-2">
        <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
          <label class="flex min-h-13 items-center justify-between gap-4 p-4 text-sm font-medium">
            {{ $t("settings.compact") }}
            <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="compact" />
          </label>
          <label class="flex min-h-13 items-center justify-between gap-4 p-4 text-sm font-medium">
            {{ $t("settings.small-scrollbars") }}
            <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="smallerScrollbars" />
          </label>
          <label class="flex min-h-13 items-center justify-between gap-4 p-4 text-sm font-medium">
            {{ $t("settings.show-timestamps") }}
            <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="showTimestamp" />
          </label>
          <label class="flex min-h-13 items-center justify-between gap-4 p-4 text-sm font-medium">
            {{ $t("settings.show-std") }}
            <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="showStd" />
          </label>
          <label class="flex min-h-13 items-center justify-between gap-4 p-4 text-sm font-medium">
            {{ $t("settings.soft-wrap") }}
            <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="softWrap" />
          </label>
          <div class="flex min-h-13 flex-wrap items-center justify-between gap-3 p-4 text-sm font-medium">
            <span>{{ $t("settings.datetime-format") }}</span>
            <div class="ml-auto flex gap-1.5">
              <DropdownMenu
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
                v-model="hourStyle"
                :options="[
                  { label: $t('settings.hour.auto'), value: 'auto' },
                  { label: $t('settings.hour.12'), value: '12' },
                  { label: $t('settings.hour.24'), value: '24' },
                ]"
              />
            </div>
          </div>
          <div class="flex min-h-13 flex-wrap items-center justify-between gap-3 p-4 text-sm font-medium">
            <span>{{ $t("settings.font-size") }}</span>
            <div class="join ml-auto">
              <button
                v-for="opt in [
                  { label: $t('settings.size.small'), value: 'small' },
                  { label: $t('settings.size.medium'), value: 'medium' },
                  { label: $t('settings.size.large'), value: 'large' },
                ]"
                :key="opt.value"
                class="btn btn-sm join-item"
                :class="size === opt.value ? 'btn-primary' : 'btn-ghost'"
                @click="size = opt.value as typeof size"
              >
                {{ opt.label }}
              </button>
            </div>
          </div>
        </div>

        <LogList
          :messages="fakeMessages"
          :last-selected-item="undefined"
          :show-container-name="false"
          class="border-base-content/15 hidden h-full overflow-hidden rounded-lg border @3xl:block"
        />
      </div>
    </section>

    <!-- OPTIONS -->
    <section class="flex flex-col gap-4">
      <div>
        <h2 class="text-xl font-semibold tracking-tight">{{ $t("settings.options") }}</h2>
        <p class="text-base-content/60 mt-1 text-sm">{{ $t("settings.options-desc") }}</p>
      </div>

      <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
        <div class="flex min-h-13 flex-wrap items-center justify-between gap-3 p-4 text-sm font-medium">
          <span>{{ $t("settings.locale") }}</span>
          <DropdownMenu
            class="ml-auto"
            v-model="locale"
            :options="[
              { label: 'Auto', value: '' },
              ...availableLocales.map((l) => ({ label: l.toLocaleUpperCase(), value: l })),
            ]"
          />
        </div>
        <div class="flex min-h-13 flex-wrap items-center justify-between gap-3 p-4 text-sm font-medium">
          <span>{{ $t("settings.color-scheme") }}</span>
          <div class="join ml-auto">
            <button
              v-for="opt in [
                { label: $t('settings.theme.light'), value: 'light' },
                { label: $t('settings.theme.dark'), value: 'dark' },
                { label: $t('settings.theme.auto'), value: 'auto' },
              ]"
              :key="opt.value"
              class="btn btn-sm join-item"
              :class="lightTheme === opt.value ? 'btn-primary' : 'btn-ghost'"
              @click="lightTheme = opt.value as typeof lightTheme"
            >
              {{ opt.label }}
            </button>
          </div>
        </div>
        <div class="flex min-h-13 flex-wrap items-center justify-between gap-3 p-4 text-sm font-medium">
          <span>{{ $t("settings.automatic-redirect") }}</span>
          <DropdownMenu
            class="ml-auto"
            v-model="automaticRedirect"
            :options="[
              { label: $t('settings.redirect.instant'), value: 'instant' },
              { label: $t('settings.redirect.delayed'), value: 'delayed' },
              { label: $t('settings.redirect.none'), value: 'none' },
            ]"
          />
        </div>
        <div class="flex min-h-13 flex-wrap items-center justify-between gap-3 p-4 text-sm font-medium">
          <span>{{ $t("settings.group-containers") }}</span>
          <DropdownMenu
            class="ml-auto"
            v-model="groupContainers"
            :options="[
              { label: $t('settings.grouping.always'), value: 'always' },
              { label: $t('settings.grouping.at-least-2'), value: 'at-least-2' },
              { label: $t('settings.grouping.never'), value: 'never' },
            ]"
          />
        </div>
        <label class="flex min-h-13 items-center justify-between gap-4 p-4 text-sm font-medium">
          <span>{{ $t("settings.search") }} <key-shortcut char="f" class="align-top"></key-shortcut></span>
          <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="search" />
        </label>
        <label class="flex min-h-13 items-center justify-between gap-4 p-4 text-sm font-medium">
          {{ $t("settings.show-stopped-containers") }}
          <input type="checkbox" class="toggle toggle-primary toggle-sm" v-model="showAllContainers" />
        </label>
      </div>
    </section>
  </div>
</template>

<script lang="ts" setup>
import { ComplexLogEntry, SimpleLogEntry, GroupedLogEntry } from "@/models/LogEntry";

import {
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
} from "@/stores/settings";

import { availableLocales, i18n } from "@/modules/i18n";

const { t } = useI18n();

setTitle(t("title.settings"));
const { latestRelease, hasRelease } = useAnnouncements();

const dateFmt: Intl.DateTimeFormatOptions = { year: "numeric", month: "short", day: "numeric" };

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

:deep(.text-base-content\/60 a:not(.btn)),
:deep(.text-base-content\/70 a:not(.btn)) {
  @apply text-primary;
}
:deep(.text-base-content\/60 a:not(.btn):hover),
:deep(.text-base-content\/70 a:not(.btn):hover) {
  text-decoration: underline;
  text-underline-offset: 4px;
}
</style>
