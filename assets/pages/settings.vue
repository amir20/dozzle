<template>
  <PageWithLinks>
    <section>
      <div class="has-underline">
        <h2>{{ $t("settings.about") }}</h2>
      </div>

      <div class="flex flex-row gap-2">
        <span v-html="$t('settings.using-version', { version: config.version })"></span>
        <span
          v-if="hasRelease"
          v-html="$t('settings.update-available', { nextVersion: latestRelease?.name, href: latestRelease?.htmlUrl })"
        ></span>
      </div>

      <div class="mt-4">
        {{ $t("settings.help-support") }}

        <ul class="mt-6 flex gap-2">
          <li>
            <a href="https://github.com/amir20/dozzle" target="_blank" rel="noopener noreferrer" class="btn">
              <mdi:github /> amir20/dozzle
            </a>
          </li>
          <li>
            <a
              href="https://buymeacoffee.com/amirraminfar"
              target="_blank"
              rel="noopener noreferrer"
              class="btn btn-secondary"
            >
              <mdi:beer />
              Buy me a beer
            </a>
          </li>
        </ul>
      </div>
    </section>

    <section class="@container flex flex-col">
      <div class="has-underline">
        <h2>{{ $t("settings.display") }}</h2>
      </div>

      <section class="grid-cols-2 gap-4 @3xl:grid">
        <div class="flex flex-col gap-4 text-balance @3xl:pr-8">
          <Toggle v-model="compact"> {{ $t("settings.compact") }} </Toggle>

          <Toggle v-model="smallerScrollbars"> {{ $t("settings.small-scrollbars") }} </Toggle>

          <Toggle v-model="showTimestamp">{{ $t("settings.show-timesamps") }}</Toggle>

          <Toggle v-model="showStd">{{ $t("settings.show-std") }}</Toggle>

          <Toggle v-model="softWrap">{{ $t("settings.soft-wrap") }}</Toggle>

          <LabeledInput>
            <template #label>
              {{ $t("settings.locale") }}
            </template>
            <template #input>
              <DropdownMenu
                v-model="locale"
                :options="[
                  { label: 'Auto', value: '' },
                  ...availableLocales.map((l) => ({ label: l.toLocaleUpperCase(), value: l })),
                ]"
              />
            </template>
          </LabeledInput>

          <LabeledInput>
            <template #label>
              {{ $t("settings.datetime-format") }}
            </template>
            <template #input>
              <div class="flex gap-4">
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
                    { label: 'Auto', value: 'auto' },
                    { label: '12', value: '12' },
                    { label: '24', value: '24' },
                  ]"
                />
              </div>
            </template>
          </LabeledInput>

          <LabeledInput>
            <template #label>
              {{ $t("settings.font-size") }}
            </template>
            <template #input>
              <DropdownMenu
                v-model="size"
                :options="[
                  { label: 'Small', value: 'small' },
                  { label: 'Medium', value: 'medium' },
                  { label: 'Large', value: 'large' },
                ]"
              />
            </template>
          </LabeledInput>

          <LabeledInput>
            <template #label>
              {{ $t("settings.color-scheme") }}
            </template>
            <template #input>
              <DropdownMenu
                v-model="lightTheme"
                :options="[
                  { label: 'Auto', value: 'auto' },
                  { label: 'Dark', value: 'dark' },
                  { label: 'Light', value: 'light' },
                ]"
              />
            </template>
          </LabeledInput>
        </div>
        <LogList
          :messages="fakeMessages"
          :last-selected-item="undefined"
          :show-container-name="false"
          class="border-base-content/50 hidden overflow-hidden rounded-lg border shadow-sm @3xl:block"
        />
      </section>
    </section>

    <section class="flex flex-col gap-4">
      <div class="has-underline">
        <h2>{{ $t("settings.options") }}</h2>
      </div>

      <LabeledInput>
        <template #label>
          {{ $t("settings.automatic-redirect") }}
        </template>
        <template #input>
          <DropdownMenu
            v-model="automaticRedirect"
            :options="[
              { label: 'Instant', value: 'instant' },
              { label: 'Delayed', value: 'delayed' },
              { label: 'None', value: 'none' },
            ]"
          />
        </template>
      </LabeledInput>

      <Toggle v-model="search">
        {{ $t("settings.search") }} <key-shortcut char="f" class="align-top"></key-shortcut>
      </Toggle>

      <Toggle v-model="showAllContainers">{{ $t("settings.show-stopped-containers") }}</Toggle>
    </section>
  </PageWithLinks>
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

import { availableLocales, i18n } from "@/modules/i18n";

const { t } = useI18n();

setTitle(t("title.settings"));
const { latestRelease, hasRelease } = useAnnouncements();

const now = new Date();
const hoursAgo = (hours: number) => {
  const date = new Date(now);
  date.setHours(date.getHours() - hours);
  return date;
};

const fakeMessages = computedWithControl(
  () => i18n.global.locale.value,
  () => [
    new SimpleLogEntry(t("settings.log.preview"), "123", 1, hoursAgo(16), "info", undefined, "stdout", ""),
    new SimpleLogEntry(t("settings.log.warning"), "123", 2, hoursAgo(12), "warn", undefined, "stdout", ""),
    new SimpleLogEntry(
      t("settings.log.multi-line-error.start-line"),
      "123",
      3,
      hoursAgo(7),
      "error",
      "start",
      "stderr",
      "",
    ),
    new SimpleLogEntry(
      t("settings.log.multi-line-error.middle-line"),
      "123",
      4,
      hoursAgo(2),
      "error",
      "middle",
      "stderr",
      "",
    ),
    new SimpleLogEntry(t("settings.log.multi-line-error.end-line"), "123", 5, new Date(), "error", "end", "stderr", ""),
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
    new SimpleLogEntry(t("settings.log.simple"), "123", 7, new Date(), "debug", undefined, "stderr", ""),
  ],
);
</script>
<style scoped>
@reference "@/main.css";

.has-underline {
  @apply border-base-content/50 mb-4 border-b py-2;

  h2 {
    @apply text-3xl;
  }
}

:deep(a:not(.menu a):not(.btn)) {
  @apply text-primary underline-offset-4 hover:underline;
}
</style>
