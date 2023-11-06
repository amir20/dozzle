<template>
  <page-with-links>
    <section>
      <div class="has-underline">
        <h2>{{ $t("settings.about") }}</h2>
      </div>

      <div>
        <span v-html="$t('settings.using-version', { version: currentVersion })"></span>
        <div
          v-if="hasUpdate"
          v-html="$t('settings.update-available', { nextVersion: nextRelease.name, href: nextRelease.html_url })"
        ></div>
      </div>
    </section>

    <section class="flex flex-col gap-4">
      <div class="has-underline">
        <h2>{{ $t("settings.display") }}</h2>
      </div>

      <div>
        <toggle v-model="smallerScrollbars"> {{ $t("settings.small-scrollbars") }} </toggle>
      </div>
      <div>
        <toggle v-model="showTimestamp">{{ $t("settings.show-timesamps") }}</toggle>
      </div>
      <div>
        <toggle v-model="showStd">{{ $t("settings.show-std") }}</toggle>
      </div>

      <div>
        <toggle v-model="softWrap">{{ $t("settings.soft-wrap") }}</toggle>
      </div>

      <div class="flex items-center gap-6">
        <dropdown
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
        <dropdown
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
        <dropdown
          v-model="lightTheme"
          :options="[
            { label: 'Auto', value: 'auto' },
            { label: 'Dark', value: 'dark' },
            { label: 'Light', value: 'light' },
          ]"
        />
        {{ $t("settings.color-scheme") }}
      </div>
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
import {
  search,
  lightTheme,
  smallerScrollbars,
  showTimestamp,
  showStd,
  hourStyle,
  showAllContainers,
  size,
  softWrap,
  automaticRedirect,
} from "@/stores/settings";

const { t } = useI18n();

setTitle(t("title.settings"));

const currentVersion = config.version;
let nextRelease = $ref({ html_url: "", name: "" });
let hasUpdate = $ref(false);

async function fetchNextRelease() {
  if (!["dev", "master"].includes(currentVersion)) {
    const response = await fetch("https://api.github.com/repos/amir20/dozzle/releases/latest");
    if (response.ok) {
      const release = await response.json();
      hasUpdate =
        release.tag_name.slice(1).localeCompare(currentVersion, undefined, { numeric: true, sensitivity: "base" }) > 0;
      nextRelease = release;
    }
  } else {
    hasUpdate = true;
    nextRelease = {
      html_url: "",
      name: "master",
    };
  }
}

fetchNextRelease();
</script>
<style lang="postcss" scoped>
.has-underline {
  @apply mb-4 border-b border-base-content/50 py-4;
}

:deep(a:not(.menu a)) {
  @apply text-primary underline-offset-4 hover:underline;
}
</style>
