<template>
  <page-with-links>
    <section>
      <div class="has-underline">
        <h2>{{ $t("settings.about") }}</h2>
      </div>

      <div>
        <span v-html="$t('settings.using-version', { version: config.version })"></span>
        <div
          v-if="hasUpdate"
          v-html="$t('settings.update-available', { nextVersion: latest?.name, href: latest?.htmlUrl })"
        ></div>
      </div>
    </section>

    <section class="flex flex-col @container">
      <div class="has-underline">
        <h2>{{ $t("settings.display") }}</h2>
      </div>

      <section class="grid-cols-2 gap-4 @3xl:grid">
        <div class="flex flex-col gap-2 text-balance @3xl:pr-8">
          <toggle v-model="compact"> {{ $t("settings.compact") }} </toggle>

          <toggle v-model="smallerScrollbars"> {{ $t("settings.small-scrollbars") }} </toggle>

          <toggle v-model="showTimestamp">{{ $t("settings.show-timesamps") }}</toggle>

          <toggle v-model="showStd">{{ $t("settings.show-std") }}</toggle>

          <toggle v-model="softWrap">{{ $t("settings.soft-wrap") }}</toggle>

          <labeled-input>
            <template #label>
              {{ $t("settings.locale") }}
            </template>
            <template #input>
              <dropdown-menu
                v-model="locale"
                :options="[
                  { label: 'Auto', value: '' },
                  ...availableLocales.map((l) => ({ label: l.toLocaleUpperCase(), value: l })),
                ]"
              />
            </template>
          </labeled-input>

          <labeled-input>
            <template #label>
              {{ $t("settings.datetime-format") }}
            </template>
            <template #input>
              <div class="flex gap-2">
                <dropdown-menu
                  v-model="dateLocale"
                  :options="[
                    { label: 'Auto', value: 'auto' },
                    { label: 'MM/DD/YYYY', value: 'en-US' },
                    { label: 'DD/MM/YYYY', value: 'en-GB' },
                    { label: 'DD.MM.YYYY', value: 'de-DE' },
                    { label: 'YYYY-MM-DD', value: 'en-CA' },
                  ]"
                />
                <dropdown-menu
                  v-model="hourStyle"
                  :options="[
                    { label: 'Auto', value: 'auto' },
                    { label: '12', value: '12' },
                    { label: '24', value: '24' },
                  ]"
                />
              </div>
            </template>
          </labeled-input>

          <labeled-input>
            <template #label>
              {{ $t("settings.font-size") }}
            </template>
            <template #input>
              <dropdown-menu
                v-model="size"
                :options="[
                  { label: 'Small', value: 'small' },
                  { label: 'Medium', value: 'medium' },
                  { label: 'Large', value: 'large' },
                ]"
              />
            </template>
          </labeled-input>

          <labeled-input>
            <template #label>
              {{ $t("settings.color-scheme") }}
            </template>
            <template #input>
              <dropdown-menu
                v-model="lightTheme"
                :options="[
                  { label: 'Auto', value: 'auto' },
                  { label: 'Dark', value: 'dark' },
                  { label: 'Light', value: 'light' },
                ]"
              />
            </template>
          </labeled-input>
        </div>
        <log-viewer
          :messages="fakeMessages"
          :visible-keys="keys"
          :last-selected-item="undefined"
          class="hidden overflow-hidden rounded-lg border border-base-content/50 shadow @3xl:block"
        />
      </section>
    </section>

    <section class="flex flex-col gap-2">
      <div class="has-underline">
        <h2>{{ $t("settings.options") }}</h2>
      </div>
      <div>
        <toggle v-model="search">
          {{ $t("settings.search") }} <key-shortcut char="f" class="align-top"></key-shortcut>
        </toggle>
      </div>

      <div>
        <toggle v-model="showAllContainers">{{ $t("settings.show-stopped-containers") }}</toggle>
      </div>

      <div>
        <toggle v-model="automaticRedirect">{{ $t("settings.automatic-redirect") }}</toggle>
      </div>
    </section>
  </page-with-links>
</template>

<script lang="ts" setup>
import { ComplexLogEntry, SimpleLogEntry } from "@/models/LogEntry";

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
} from "@/stores/settings";

const { t, availableLocales } = useI18n();

setTitle(t("title.settings"));
const { latest, hasUpdate } = useReleases();

const keys = ref<string[][]>([]);
const hoursAgo = (hours: number) => {
  const date = new Date();
  date.setHours(date.getHours() - hours);
  return date;
};

const fakeMessages = [
  new SimpleLogEntry("This is a preview of the logs", 1, hoursAgo(16), "info", undefined, "stdout"),
  new SimpleLogEntry("A warning log looks like this", 2, hoursAgo(12), "warn", undefined, "stdout"),
  new SimpleLogEntry("This is a multi line error message", 3, hoursAgo(7), "error", "start", "stderr"),
  new SimpleLogEntry("with a second line", 4, hoursAgo(2), "error", "middle", "stderr"),
  new SimpleLogEntry("and finally third line.", 5, new Date(), "error", "end", "stderr"),
  new ComplexLogEntry(
    {
      message: "This is a complex log entry as json",
      context: {
        key: "value",
        key2: "value2",
      },
    },
    6,
    new Date(),
    "info",
    "stdout",
    keys,
  ),
  new SimpleLogEntry(
    "This is a very very long message which would wrap by default. Disabling soft wraps would disable this. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. ",
    7,
    new Date(),
    "debug",
    undefined,
    "stderr",
  ),
];
</script>
<style lang="postcss" scoped>
.has-underline {
  @apply mb-4 border-b border-base-content/50 py-4;
}

:deep(a:not(.menu a)) {
  @apply text-primary underline-offset-4 hover:underline;
}
</style>
