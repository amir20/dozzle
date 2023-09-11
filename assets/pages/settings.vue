<template>
  <div class="mt-10 flex flex-col gap-8 px-10">
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

    <section>
      <div class="has-underline">
        <h2>{{ $t("settings.display") }}</h2>
      </div>

      <div class="item">
        <toggle v-model="smallerScrollbars"> {{ $t("settings.small-scrollbars") }} </toggle>
      </div>
      <div class="item">
        <toggle v-model="showTimestamp">{{ $t("settings.show-timesamps") }}</toggle>
      </div>
      <div class="item">
        <toggle v-model="showStd">{{ $t("settings.show-std") }}</toggle>
      </div>

      <div class="item">
        <toggle v-model="softWrap">{{ $t("settings.soft-wrap") }}</toggle>
      </div>

      <div class="item">
        <div class="columns is-vcentered">
          <div class="column is-narrow">
            <o-field>
              <dropdown
                v-model="hourStyle"
                :options="[
                  { label: 'Auto', value: 'auto' },
                  { label: '12', value: '12' },
                  { label: '24', value: '24' },
                ]"
              />
            </o-field>
          </div>
          <div class="column">
            {{ $t("settings.12-24-format") }}
          </div>
        </div>
      </div>
      <div class="item">
        <div class="columns is-vcentered">
          <div class="column is-narrow">
            <o-field>
              <dropdown
                v-model="size"
                :options="[
                  { label: 'Small', value: 'small' },
                  { label: 'Medium', value: 'medium' },
                  { label: 'Large', value: 'large' },
                ]"
              />
            </o-field>
          </div>
          <div class="column">{{ $t("settings.font-size") }}</div>
        </div>
      </div>
      <div class="item">
        <div class="columns is-vcentered">
          <div class="column is-narrow">
            <o-field>
              <dropdown
                v-model="lightTheme"
                :options="[
                  { label: 'Auto', value: 'auto' },
                  { label: 'Dark', value: 'dark' },
                  { label: 'Light', value: 'light' },
                ]"
              />
            </o-field>
          </div>
          <div class="column">{{ $t("settings.color-scheme") }}</div>
        </div>
      </div>
    </section>
    <section>
      <div class="has-underline">
        <h2>{{ $t("settings.options") }}</h2>
      </div>
      <div class="item">
        <toggle v-model="search">
          <div>{{ $t("settings.search") }} <key-shortcut char="f" class="align-top"></key-shortcut></div>
        </toggle>
      </div>

      <div class="item">
        <toggle v-model="showAllContainers">{{ $t("settings.show-stopped-containers") }}</toggle>
      </div>
    </section>
  </div>
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
} from "@/composables/settings";

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
a.next-release {
  text-decoration: underline;

  &:hover {
    text-decoration: none;
  }
}

.has-underline {
  @apply mb-4 border-b border-scheme-inverted py-4;

  h2 {
    @apply text-3xl;
  }
}

.item {
  padding: 1em 0;
}

code {
  border-radius: 4px;
  background-color: #444;
}
</style>
