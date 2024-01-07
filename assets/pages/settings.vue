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

    <section class="flex flex-col">
      <div class="has-underline">
        <h2>{{ $t("settings.display") }}</h2>
      </div>

      <section class="grid grid-cols-2 gap-4">
        <div class="flex flex-col items-start gap-4 text-balance">
          <toggle v-model="smallerScrollbars"> {{ $t("settings.small-scrollbars") }} </toggle>

          <toggle v-model="showTimestamp">{{ $t("settings.show-timesamps") }}</toggle>

          <toggle v-model="showStd">{{ $t("settings.show-std") }}</toggle>

          <toggle v-model="softWrap">{{ $t("settings.soft-wrap") }}</toggle>

          <div class="flex items-center gap-6">
            <dropdown-menu
              v-model="hourStyle"
              :options="[
                { label: 'Auto', value: 'auto' },
                { label: '12', value: '12' },
                { label: '24', value: '24' },
              ]"
            />
            {{ $t("settings.12-24-format") }}
          </div>
          <div class="flex items-center gap-6">
            <dropdown-menu
              v-model="size"
              :options="[
                { label: 'Small', value: 'small' },
                { label: 'Medium', value: 'medium' },
                { label: 'Large', value: 'large' },
              ]"
            />
            {{ $t("settings.font-size") }}
          </div>
          <div class="flex items-center gap-6">
            <dropdown-menu
              v-model="lightTheme"
              :options="[
                { label: 'Auto', value: 'auto' },
                { label: 'Dark', value: 'dark' },
                { label: 'Light', value: 'light' },
              ]"
            />
            {{ $t("settings.color-scheme") }}
          </div>
        </div>
        <log-viewer
          :messages="fakeMessages"
          :visible-keys="[]"
          :last-selected-item="undefined"
          class="rounded border border-base-content/50 shadow"
        />
      </section>
    </section>

    <section class="flex flex-col gap-2">
      <div class="has-underline">
        <h2>{{ $t("settings.options") }}</h2>
      </div>
      <div>
        <toggle v-model="search">
          <div>{{ $t("settings.search") }} <key-shortcut char="f" class="align-top"></key-shortcut></div>
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
import { SimpleLogEntry } from "@/models/LogEntry";
import {
  automaticRedirect,
  hourStyle,
  lightTheme,
  search,
  showAllContainers,
  showStd,
  showTimestamp,
  size,
  smallerScrollbars,
  softWrap,
} from "@/stores/settings";

const { t } = useI18n();

setTitle(t("title.settings"));
const { latest, hasUpdate } = useReleases();

const fakeMessages = [
  new SimpleLogEntry("This is a test message", 1, new Date(), "info", undefined, "stdout"),
  new SimpleLogEntry("This is a test error", 1, new Date(), "error", undefined, "stdout"),
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
