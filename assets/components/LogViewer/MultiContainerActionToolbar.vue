<template>
  <div class="dropdown dropdown-end dropdown-hover z-20">
    <label tabindex="0" class="icon-btn btn btn-ghost btn-sm w-8 gap-0 px-0 md:gap-0.5">
      <carbon:circle-solid class="text-red w-2 md:w-2.5" v-if="streamConfig.stderr" />
      <carbon:circle-solid class="text-blue w-2 md:w-2.5" v-if="streamConfig.stdout" />
    </label>
    <ul
      tabindex="0"
      class="menu dropdown-content rounded-box bg-base-200 border-base-content/10 z-50 w-max min-w-60 border p-1.5 shadow-lg"
      @click="hideMenu"
    >
      <li class="section">{{ $t("toolbar.section-logs") }}</li>
      <li>
        <a @click="showSearch = true">
          <mdi:magnify /> {{ $t("toolbar.search") }}
          <KeyShortcut char="f" />
        </a>
      </li>
      <li>
        <a @click="clear()">
          <octicon:trash-24 /> {{ $t("toolbar.clear") }}
          <KeyShortcut char="l" :modifiers="['shift', 'meta']" />
        </a>
      </li>

      <li class="section">{{ $t("toolbar.section-filters") }}</li>
      <li>
        <details>
          <summary>
            <div class="flex w-4 items-center gap-0.5">
              <carbon:circle-solid class="size-2" :class="streamConfig.stderr ? 'text-red' : 'opacity-20'" />
              <carbon:circle-solid class="size-2" :class="streamConfig.stdout ? 'text-blue' : 'opacity-20'" />
            </div>
            {{ $t("toolbar.streams") }}
            <span class="value">{{ streamSummary }}</span>
          </summary>
          <ul class="menu">
            <li>
              <a
                @click="
                  streamConfig.stdout = true;
                  streamConfig.stderr = true;
                "
              >
                <mdi:check class="w-4" v-if="streamConfig.stdout == true && streamConfig.stderr == true" />
                <div v-else class="w-4"></div>
                {{ $t("toolbar.show-all") }}
              </a>
            </li>
            <li>
              <a
                @click="
                  streamConfig.stdout = true;
                  streamConfig.stderr = false;
                "
              >
                <mdi:check class="w-4" v-if="streamConfig.stdout == true && streamConfig.stderr == false" />
                <div v-else class="w-4"></div>
                {{ $t("toolbar.show", { std: "STDOUT" }) }}
              </a>
            </li>
            <li>
              <a
                @click="
                  streamConfig.stdout = false;
                  streamConfig.stderr = true;
                "
              >
                <mdi:check class="w-4" v-if="streamConfig.stdout == false && streamConfig.stderr == true" />
                <div v-else class="w-4"></div>
                {{ $t("toolbar.show", { std: "STDERR" }) }}
              </a>
            </li>
          </ul>
        </details>
      </li>
      <li>
        <details class="group/details">
          <summary>
            <mdi:gauge />
            {{ $t("toolbar.levels") }}
            <span class="value group-open/details:hidden">{{ levelSummary }}</span>
            <Toggle
              class="toggle-xs hidden group-open/details:inline-flex"
              v-model="toggleAllLevels"
              :title="$t('toolbar.toggle-all-levels')"
            />
          </summary>
          <ul class="menu">
            <li v-for="level in allLevels">
              <a class="capitalize" @click="levels.has(level) ? levels.delete(level) : levels.add(level)">
                <mdi:check class="w-4" v-if="levels.has(level)" />
                <div v-else class="w-4"></div>

                <div class="flex">
                  <div class="badge" :data-level="level">{{ level }}</div>
                </div>
              </a>
            </li>
          </ul>
        </details>
      </li>

      <!-- Only a merged stream mixes containers and hosts in one column, so these
           two toggles have nowhere else to live. -->
      <li class="section">{{ $t("toolbar.section-display") }}</li>
      <li>
        <a @click="showContainerName = !showContainerName">
          <mdi:check class="w-4" v-if="showContainerName" />
          <div v-else class="w-4"></div>
          {{ $t("toolbar.show-container-name") }}
        </a>
      </li>
      <li>
        <a @click="showHostname = !showHostname">
          <mdi:check class="w-4" v-if="showHostname" />
          <div v-else class="w-4"></div>
          {{ $t("toolbar.show-hostname") }}
        </a>
      </li>

      <template v-if="enableDownload">
        <li class="section">{{ $t("toolbar.section-export") }}</li>
        <li>
          <a :href="downloadUrl" download>
            <octicon:download-24 />
            {{ isFiltered ? $t("toolbar.download-filtered") : $t("toolbar.download") }}
          </a>
        </li>
      </template>
    </ul>
  </div>
</template>

<script lang="ts" setup>
import { allLevels } from "@/composable/logContext";

const { showSearch } = useSearchFilter();
const { enableDownload } = config;
const clear = defineEmit();
const { t } = useI18n();

const { name } = defineProps<{ name?: string }>();

const { streamConfig, showHostname, showContainerName, containers, levels } = useLoggingContext();

const { downloadUrl, isFiltered } = useDownloadUrl(containers, streamConfig, levels, name);

// Collapsed submenus say what they are currently set to, so the menu answers
// "what am I looking at?" without being opened.
const streamSummary = computed(() => {
  if (streamConfig.value.stdout && streamConfig.value.stderr) return t("toolbar.all");
  if (streamConfig.value.stdout) return "STDOUT";
  if (streamConfig.value.stderr) return "STDERR";
  return t("toolbar.none");
});

const levelSummary = computed(() =>
  levels.value.size === allLevels.length ? t("toolbar.all") : `${levels.value.size}/${allLevels.length}`,
);

const toggleAllLevels = computed({
  get: () => levels.value.size === allLevels.length,
  set: (value) => {
    if (value) {
      allLevels.forEach((level) => levels.value.add(level));
    } else {
      levels.value.clear();
    }
  },
});

const hideMenu = (e: MouseEvent) => {
  if (e.target instanceof HTMLAnchorElement) {
    setTimeout(() => {
      if (document.activeElement instanceof HTMLElement) {
        document.activeElement.blur();
      }
    }, 50);
  }
};
</script>

<style scoped>
@reference "@/main.css";

/* Section headers replace the old hairline dividers: they separate and label at
 * the same time. The first one has nothing above it to divide from. */
li.section {
  @apply text-base-content/40 border-base-content/10 mt-1.5 border-t px-2 pt-2 pb-1 text-[10px] font-semibold tracking-wider uppercase;
}

/* daisyUI pads every direct child of a menu li as if it were a clickable row,
 * which squeezes anything sitting inside these headers down to nothing. */
li.section > * {
  @apply p-0;
}

li.section:first-child {
  @apply mt-0 border-t-0 pt-1;
}

a,
button,
summary {
  @apply whitespace-nowrap;
}

/* One icon size for every row, so labels line up on a single text column. */
.menu > li > :where(a, button, summary) > svg {
  @apply size-4 shrink-0 opacity-70;
}

/* The collapsed-state value sits in the trailing grid column daisyUI reserves,
 * next to (not on top of) the disclosure chevron. */
.value {
  @apply text-base-content/40 text-xs tabular-nums;
}

/* daisyUI's .menu is width: fit-content, so nested submenus (Streams, Levels)
 * shrink to their content and the hover highlight stops short. Stretch them to
 * fill the dropdown so the row highlight spans the full width. */
.menu li ul {
  margin-inline-start: 0;
  width: 100%;
  &:before {
    display: none;
  }
}

/* Keep the solid level colors, but use white labels in the light theme so the
 * text reads against the saturated chip backgrounds. warn is a light orange,
 * where dark text has better contrast than white, so it keeps the default. */
[data-theme="light"] .badge[data-level="info"],
[data-theme="light"] .badge[data-level="debug"],
[data-theme="light"] .badge[data-level="trace"],
[data-theme="light"] .badge[data-level="error"],
[data-theme="light"] .badge[data-level="fatal"] {
  color: oklch(100% 0 0) !important;
}
</style>
