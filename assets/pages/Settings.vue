<template>
  <div>
    <section class="section">
      <div class="has-underline">
        <h2 class="title is-4">About</h2>
      </div>

      <div>
        You are using Dozzle <i>{{ currentVersion }}</i
        >.
        <span v-if="hasUpdate">
          New version is available! Update to
          <a :href="nextRelease.html_url" class="next-release" target="_blank" rel="noreferrer noopener">
            {{ nextRelease.name }}</a
          >.
        </span>
      </div>
    </section>

    <section class="section">
      <div class="has-underline">
        <h2 class="title is-4">Display</h2>
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
            By default, Dozzle will use your browser's locale to format time. You can force to 12 or 24 hour style.
          </div>
        </div>

        <div class="item">
          <o-switch v-model="smallerScrollbars"> Use smaller scrollbars </o-switch>
        </div>
        <div class="item">
          <o-switch v-model="showTimestamp"> Show timestamps </o-switch>
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
          <div class="column">Font size to use for logs</div>
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
          <div class="column">Color scheme</div>
        </div>
      </div>
    </section>
    <section class="section">
      <div class="has-underline">
        <h2 class="title is-4">Options</h2>
      </div>

      <div class="item">
        <o-switch v-model="search">
          Enable searching with Dozzle using <code>command+f</code> or <code>ctrl+f</code>
        </o-switch>
      </div>

      <div class="item">
        <o-switch v-model="showAllContainers"> Show stopped containers </o-switch>
      </div>
    </section>
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import gt from "semver/functions/gt";
import config from "@/stores/config";
import { setTitle } from "@/composables/title";
import {
  search,
  lightTheme,
  smallerScrollbars,
  showTimestamp,
  hourStyle,
  showAllContainers,
  size,
} from "@/composables/settings";

setTitle("Settings");

const currentVersion = config.version;
const nextRelease = ref({ html_url: "", name: "" });
const hasUpdate = ref(false);

async function fetchNextRelease() {
  if (!["dev", "master"].includes(currentVersion)) {
    const response = await fetch("https://api.github.com/repos/dozzle/dozzle/releases/latest");
    if (response.ok) {
      const releases = await response.json();
      hasUpdate.value = gt(releases[0].tag_name, currentVersion);
      nextRelease.value = releases[0];
    }
  } else {
    hasUpdate.value = true;
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
