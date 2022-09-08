<template>
  <div>
    <section class="section">
      <div class="has-underline">
        <h2 class="title is-4">{{ $t("settings.about") }}</h2>
      </div>

      <div>
        <span v-html="$t('settings.using-version', { version: currentVersion })"></span>
        <div
          v-if="hasUpdate"
          v-html="$t('settings.update-available', { nextVersion: nextRelease.name, href: nextRelease.html_url })"
        ></div>
      </div>
    </section>

    <section class="section">
      <div class="has-underline">
        <h2 class="title is-4">{{ $t("settings.display") }}</h2>
      </div>

      <div class="item">
        <o-switch v-model="smallerScrollbars"> {{ $t("settings.small-scrollbars") }} </o-switch>
      </div>
      <div class="item">
        <o-switch v-model="showTimestamp"> {{ $t("settings.show-timesamps") }} </o-switch>
      </div>

      <div class="item">
        <o-switch v-model="softWrap"> {{ $t("settings.soft-wrap") }}</o-switch>
      </div>

      <div class="item">
        <div class="columns is-vcentered">
          <div class="column is-narrow">
            <o-field>
              <o-dropdown v-model="hourStyle" aria-role="list">
                <template #trigger>
                  <o-button variant="primary" type="button">
                    <span class="is-capitalized">{{ hourStyle }}</span>
                    <span class="icon">
                      <carbon-caret-down />
                    </span>
                  </o-button>
                </template>

                <o-dropdown-item :value="value" aria-role="listitem" v-for="value in ['auto', '12', '24']" :key="value">
                  <span class="is-capitalized">{{ value }}</span>
                </o-dropdown-item>
              </o-dropdown>
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
              <o-dropdown v-model="size" aria-role="list">
                <template #trigger>
                  <o-button variant="primary" type="button">
                    <span class="is-capitalized">{{ size }}</span>
                    <span class="icon">
                      <carbon-caret-down />
                    </span>
                  </o-button>
                </template>

                <o-dropdown-item
                  :value="value"
                  aria-role="listitem"
                  v-for="value in ['small', 'medium', 'large']"
                  :key="value"
                >
                  <span class="is-capitalized">{{ value }}</span>
                </o-dropdown-item>
              </o-dropdown>
            </o-field>
          </div>
          <div class="column">{{ $t("settings.font-size") }}</div>
        </div>
      </div>
      <div class="item">
        <div class="columns is-vcentered">
          <div class="column is-narrow">
            <o-field>
              <o-dropdown v-model="lightTheme" aria-role="list">
                <template #trigger>
                  <o-button variant="primary" type="button">
                    <span class="is-capitalized">{{ lightTheme }}</span>
                    <span class="icon">
                      <carbon-caret-down />
                    </span>
                  </o-button>
                </template>

                <o-dropdown-item
                  :value="value"
                  aria-role="listitem"
                  v-for="value in ['auto', 'dark', 'light']"
                  :key="value"
                >
                  <span class="is-capitalized">{{ value }}</span>
                </o-dropdown-item>
              </o-dropdown>
            </o-field>
          </div>
          <div class="column">{{ $t("settings.color-scheme") }}</div>
        </div>
      </div>
    </section>
    <section class="section">
      <div class="has-underline">
        <h2 class="title is-4">{{ $t("settings.options") }}</h2>
      </div>

      <div class="item">
        <o-switch v-model="search">
          <span v-html="$t('settings.search')"></span>
        </o-switch>
      </div>

      <div class="item">
        <o-switch v-model="showAllContainers"> {{ $t("settings.show-stopped-containers") }} </o-switch>
      </div>
    </section>
  </div>
</template>

<script lang="ts" setup>
import gt from "semver/functions/gt";
import {
  search,
  lightTheme,
  smallerScrollbars,
  showTimestamp,
  hourStyle,
  showAllContainers,
  size,
  softWrap,
} from "@/composables/settings";

const { t } = useI18n();

setTitle(t("title.settings"));

const currentVersion = $ref(config.version);
let nextRelease = $ref({ html_url: "", name: "" });
let hasUpdate = $ref(false);

async function fetchNextRelease() {
  if (!["dev", "master"].includes(currentVersion)) {
    const response = await fetch("https://api.github.com/repos/amir20/dozzle/releases/latest");
    if (response.ok) {
      const release = await response.json();
      hasUpdate = gt(release.tag_name, currentVersion);
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
<style lang="scss" scoped>
.title {
  color: var(--title-color);
}

a.next-release {
  text-decoration: underline;
  &:hover {
    text-decoration: none;
  }
}

.section {
  padding: 1rem 1.5rem;
}

.has-underline {
  border-bottom: 1px solid var(--border-color);
  padding: 1em 0px;
  margin-bottom: 1em;
}

.item {
  padding: 1em 0;
}

code {
  border-radius: 4px;
  background-color: #444;
}
</style>
