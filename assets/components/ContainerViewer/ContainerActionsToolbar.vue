<template>
  <Popover
    hover
    placement="bottom-end"
    panel-class="rounded-box bg-base-200 border-base-content/10 w-max min-w-60 border p-1.5 shadow-lg"
  >
    <template #trigger>
      <button type="button" class="icon-btn btn btn-ghost btn-sm relative w-8 gap-0 px-0 md:gap-0.5">
        <carbon:circle-solid class="text-red w-2 md:w-2.5" v-if="streamConfig.stderr" />
        <carbon:circle-solid class="text-blue w-2 md:w-2.5" v-if="streamConfig.stdout" />
        <span
          v-if="showImageUpdateAlert"
          class="absolute end-0.5 top-0.5 flex size-1.5"
          :title="$t('toolbar.update-available')"
        >
          <span class="bg-warning absolute size-full rounded-full opacity-75 motion-safe:animate-ping"></span>
          <span class="bg-warning relative size-full rounded-full"></span>
        </span>
      </button>
    </template>
    <ul class="menu w-full p-0">
      <li class="section" v-if="!historical || hasComplexLogs">{{ $t("toolbar.section-logs") }}</li>
      <li v-if="!historical">
        <a @click="showSearch = true">
          <mdi:magnify /> {{ $t("toolbar.search") }}
          <KeyShortcut char="f" />
        </a>
      </li>
      <!-- The title bar's pin is icon-only to keep the row short, so this is where the
           feature is actually named. Menu rows are what people scan when hunting. -->
      <li>
        <a @click="pinned = !pinned">
          <ph:map-pin-simple-fill v-if="pinned" class="text-secondary" />
          <ph:map-pin-simple v-else />
          {{ pinned ? $t("toolbar.unpin") : $t("toolbar.pin") }}
        </a>
      </li>
      <li v-if="!historical">
        <a @click="clear()">
          <octicon:trash-24 /> {{ $t("toolbar.clear") }}
          <KeyShortcut char="l" :modifiers="['shift', 'meta']" />
        </a>
      </li>
      <li v-if="hasComplexLogs">
        <a @click="showDrawer(LogAnalytics, { container }, 'lg')">
          <ph:file-sql /> {{ $t("analytics.title") }}
          <KeyShortcut char="f" :modifiers="['shift', 'meta']" />
        </a>
      </li>
      <li v-if="cloudLinked">
        <a @click="openRail('chat')">
          <mdi:message-outline /> {{ $t("cloud-chat.title") }}
          <KeyShortcut char="k" :modifiers="['shift', 'meta']" />
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

      <li class="section">{{ $t("toolbar.section-export") }}</li>
      <li v-if="enableDownload">
        <a :href="downloadUrl" download>
          <octicon:download-24 />
          {{ isFiltered ? $t("toolbar.download-filtered") : $t("toolbar.download") }}
        </a>
      </li>
      <li v-if="isSupported">
        <a @click="copyLogs()">
          <mdi:content-copy />
          {{ isFiltered ? $t("toolbar.copy-filtered-logs") : $t("toolbar.copy-logs") }}
        </a>
      </li>
      <li>
        <a @click="copyPermalink()">
          <material-symbols:link />
          {{ $t("toolbar.copy-permalink") }}
        </a>
      </li>

      <!-- Container Actions (Enabled via config) -->
      <li class="section" v-if="showContainerSection">{{ $t("toolbar.section-container") }}</li>
      <template v-if="enableActions && !historical">
        <li>
          <button
            @click="stop()"
            :disabled="actionStates.stop || actionStates.restart"
            v-if="container.state == 'running'"
          >
            <carbon:stop-filled-alt /> {{ $t("toolbar.stop") }}
          </button>

          <button
            @click="start()"
            :disabled="actionStates.start || actionStates.restart"
            v-if="container.state != 'running'"
          >
            <carbon:play /> {{ $t("toolbar.start") }}
          </button>
        </li>
        <li>
          <button @click="restart()" :disabled="disableRestart">
            <carbon:restart
              :class="{
                'animate-spin': actionStates.restart,
                'text-secondary': actionStates.restart,
              }"
            />
            {{ $t("toolbar.restart") }}
          </button>
        </li>
        <li v-if="!isSelfContainer">
          <button @click="update()" :disabled="actionStates.update">
            <carbon:upgrade />
            {{ container.isSwarm ? $t("toolbar.update-service") : $t("toolbar.update") }}
            <span v-if="showImageUpdateAlert" class="bg-warning size-1.5 rounded-full"></span>
          </button>
        </li>
      </template>

      <template v-if="enableShell && !historical">
        <li>
          <a @click="showDrawer(Terminal, { container, action: 'attach' }, 'lg')">
            <ri:terminal-window-fill />
            {{ $t("toolbar.attach") }}
            <KeyShortcut char="a" :modifiers="['shift', 'meta']" />
          </a>
        </li>
        <li>
          <a @click="showDrawer(Terminal, { container, action: 'exec' }, 'lg')">
            <material-symbols:terminal />
            {{ $t("toolbar.shell") }}
            <KeyShortcut char="e" :modifiers="['shift', 'meta']" />
          </a>
        </li>
      </template>

      <!-- Manual mode never checks on its own, so the only way to reach a
           registry is this. Automatic mode reports the last result instead. -->
      <template v-if="imageCheckState === 'check'">
        <li class="section">{{ $t("toolbar.section-image") }}</li>
        <li>
          <a @click.stop="checkImageUpdate(true)">
            <carbon:upgrade :class="{ 'animate-spin': checkingImageUpdate }" />
            {{ checkingImageUpdate ? $t("toolbar.checking-for-updates") : $t("toolbar.check-for-updates") }}
          </a>
        </li>
      </template>

      <template v-if="imageCheckState === 'checking'">
        <li class="section flex-row items-center gap-1.5">
          <carbon:upgrade class="size-3 animate-spin" /> {{ $t("toolbar.checking-for-updates") }}
        </li>
      </template>

      <template v-if="imageCheckState === 'none'">
        <li class="section flex-row items-center gap-1.5">
          <carbon:checkmark class="size-3" /> {{ $t("toolbar.no-updates") }}
        </li>
      </template>

      <!-- Shown regardless of actions: an update is worth knowing about even
           when Dozzle cannot apply it. -->
      <template v-if="imageCheckState === 'available'">
        <li class="section warn flex-row items-center gap-1.5">
          <span class="relative flex size-1.5">
            <span class="bg-warning absolute size-full rounded-full opacity-75 motion-safe:animate-ping"></span>
            <span class="bg-warning relative size-full rounded-full"></span>
          </span>
          {{ $t("toolbar.update-available") }}
        </li>
        <li v-if="isSelfContainer">
          <a :href="releaseNotesUrl" target="_blank" rel="noreferrer noopener">
            <mdi:script-text-outline /> {{ $t("toolbar.view-release-notes") }}
          </a>
        </li>
        <li>
          <a @click="copyImageReference()"> <mdi:content-copy /> {{ $t("toolbar.copy-image") }} </a>
        </li>
        <li>
          <a @click="dismissImageUpdate()"> <mdi:bell-off-outline /> {{ $t("toolbar.dismiss-update") }} </a>
        </li>
      </template>
    </ul>
  </Popover>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import { allLevels } from "@/composable/logContext";
import LogAnalytics from "../LogViewer/LogAnalytics.vue";
import Terminal from "@/components/Terminal.vue";

const { showSearch } = useSearchFilter();
const { linked: cloudLinked } = useCloudSurface();
const { openRail } = useCloudRail();
const { enableActions, enableShell, enableDownload } = config;
const { streamConfig, hasComplexLogs, levels } = useLoggingContext();
const showDrawer = useDrawer();

const { container, historical = false } = defineProps<{ container: Container; historical?: boolean }>();
const clear = defineEmit();
const { actionStates, start, stop, restart, update } = useContainerActions(toRef(() => container));
const {
  showAlert: showImageUpdateAlert,
  isSelf: isSelfContainer,
  dismiss: dismissImageUpdate,
  check: checkImageUpdate,
  checking: checkingImageUpdate,
  result: imageUpdateResult,
} = useImageUpdate(
  toRef(() => container),
  toRef(() => historical),
);
// What the menu should say about image updates. Anything not actionable
// (pinned, locally built, private registry, a failed check) shows nothing
// rather than adding a dead row to the menu.
const imageCheckState = computed(() => {
  if (config.imageCheckMode === "off" || historical) return "hidden";
  if (showImageUpdateAlert.value) return "available";
  if (config.imageCheckMode === "manual") return "check";
  if (checkingImageUpdate.value) return "checking";
  return imageUpdateResult.value?.status === "up-to-date" ? "none" : "hidden";
});

// Dozzle's own standalone container cannot restart itself, so the alert sends
// people to the release notes rather than to a button that cannot work.
const { latestRelease } = useAnnouncements();
const releaseNotesUrl = computed(() => latestRelease.value?.htmlUrl ?? "https://github.com/amir20/dozzle/releases");

const router = useRouter();
const { copy, copied, isSupported } = useClipboard({ legacy: true });
const { t } = useI18n();
const { showToast, removeToast } = useToast();

async function copyPermalink() {
  const url = router.resolve({
    name: "/show",
    query: { name: container.name, host: container.host },
  }).href;

  const resolved = new URL(url, window.location.origin);

  if (!isSupported.value) {
    showToast(
      {
        title: t("error.copy-not-supported-hint"),
        message: resolved.href,
        type: "info",
      },
      { expire: 10000 },
    );
    return;
  }

  await copy(resolved.href);

  if (copied.value) {
    showToast(
      {
        title: t("toasts.copied.title"),
        message: t("toasts.copied.message"),
        type: "info",
      },
      { expire: 2000 },
    );
  }
}

async function copyImageReference() {
  await copy(container.image);

  if (copied.value) {
    showToast(
      {
        title: t("toasts.copied.title"),
        message: escapeHtml(container.image),
        type: "info",
      },
      { expire: 2000 },
    );
  }
}

async function copyLogs() {
  const params = new URLSearchParams();
  if (streamConfig.value.stdout) params.append("stdout", "1");
  if (streamConfig.value.stderr) params.append("stderr", "1");
  params.append("everything", "1");

  const { debouncedSearchFilter } = useSearchFilter();
  if (debouncedSearchFilter.value) {
    params.append("filter", debouncedSearchFilter.value);
  }

  const selectedLevels = Array.from(levels.value);
  if (selectedLevels.length > 0 && selectedLevels.length < allLevels.length) {
    selectedLevels.forEach((level) => params.append("levels", level));
  }

  const url = withBase(`/api/hosts/${container.host}/containers/${container.id}/logs?${params.toString()}`);

  const toastId = "copy-logs";
  showToast(
    {
      id: toastId,
      title: t("toolbar.copying-logs"),
      message: "",
      type: "info",
    },
    { once: true },
  );

  const blobPromise = fetch(url, { headers: { Accept: "text/plain" } })
    .then((response) => {
      if (!response.ok) throw new Error(response.statusText);
      return response.blob();
    })
    .then((blob) => {
      removeToast(toastId);
      showToast(
        {
          title: t("toasts.copied.title"),
          message: t("toasts.copied.message"),
          type: "info",
        },
        { expire: 2000 },
      );
      return blob;
    })
    .catch((err) => {
      removeToast(toastId);
      showToast(
        {
          title: "Error",
          message: err.message,
          type: "error",
        },
        { expire: 5000 },
      );
      throw err;
    });

  await navigator.clipboard.write([new ClipboardItem({ "text/plain": blobPromise })]);
}

onKeyStroke(["f", "F"], (e) => {
  if (hasComplexLogs.value) {
    if ((e.ctrlKey || e.metaKey) && e.shiftKey) {
      showDrawer(LogAnalytics, { container }, "lg");
      e.preventDefault();
    }
  }
});
if (enableShell) {
  onKeyStroke(["a", "A"], (e) => {
    if ((e.ctrlKey || e.metaKey) && e.shiftKey) {
      showDrawer(Terminal, { container, action: "attach" }, "lg");
      e.preventDefault();
    }
  });

  onKeyStroke(["e", "E"], (e) => {
    if ((e.ctrlKey || e.metaKey) && e.shiftKey) {
      showDrawer(Terminal, { container, action: "exec" }, "lg");
      e.preventDefault();
    }
  });
}

const pinned = computed({
  get: () => pinnedContainers.value.has(container.name),
  set: (value) => {
    if (value) {
      pinnedContainers.value.add(container.name);
    } else {
      pinnedContainers.value.delete(container.name);
    }
  },
});

const containerRef = computed(() => [container]);
const { downloadUrl, isFiltered } = useDownloadUrl(
  containerRef,
  streamConfig,
  levels,
  toRef(() => container.name),
);

const disableRestart = computed(() => actionStates.stop || actionStates.start || actionStates.restart);

// The section header is shared by container actions and the shell entries, so it
// only shows when at least one of them is actually rendered.
const showContainerSection = computed(() => (enableActions || enableShell) && !historical);

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
</script>

<style scoped>
@reference "@/main.css";

/* Section headers replace the old hairline dividers: they separate and label at
 * the same time. The first one has nothing above it to divide from. */
li.section {
  @apply text-base-content/40 border-base-content/10 mt-1.5 border-t px-2 pt-2 pb-1 text-[10px] font-semibold tracking-wider uppercase;
}

/* Same specificity as li.section, declared after it, so the accent wins. */
li.section.warn {
  @apply text-warning/90;
}

/* daisyUI pads every direct child of a menu li as if it were a clickable row,
 * which squeezes the status dot and spinner in these headers down to nothing. */
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
