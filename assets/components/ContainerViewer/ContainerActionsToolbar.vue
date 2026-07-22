<template>
  <div
    class="dropdown dropdown-end z-20"
    :class="{ 'dropdown-hover': canHover }"
    @mouseenter="actionsMenuOpen = true"
    @mouseleave="onLeave"
  >
    <div
      tabindex="0"
      role="button"
      class="btn btn-ghost btn-sm w-8 px-0"
      :aria-label="$t('toolbar.more-actions')"
      @click="focusOnTap"
    >
      <mdi:dots-horizontal class="size-6" />
    </div>
    <ul
      tabindex="0"
      class="menu dropdown-content rounded-box bg-base-200 border-base-content/20 z-50 w-52 border p-1 shadow-sm"
      @click="hideMenu"
    >
      <li v-if="!historical">
        <a @click="showSearch = true">
          <mdi:magnify /> {{ $t("toolbar.search") }}
          <KeyShortcut char="f" />
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
          <ph:file-sql /> SQL Analytics
          <KeyShortcut char="f" :modifiers="['shift', 'meta']" />
        </a>
      </li>
      <li class="line"></li>
      <li>
        <details>
          <summary>
            <div class="flex w-4">
              <carbon:circle-solid class="text-red w-2.5" v-if="streamConfig.stderr" />
              <carbon:circle-solid class="text-blue w-2.5" v-if="streamConfig.stdout" />
            </div>
            Streams
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
            Levels
            <Toggle
              class="toggle-xs invisible group-open/details:visible"
              v-model="toggleAllLevels"
              title="Toggle all levels"
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

      <li class="line"></li>
      <li v-if="enableDownload">
        <a :href="downloadUrl" download>
          <octicon:download-24 />
          {{ isFiltered ? $t("toolbar.download-filtered") : $t("toolbar.download") }}
        </a>
      </li>
      <li>
        <a @click="copyPermalink()">
          <material-symbols:link />
          {{ $t("toolbar.copy-permalink") }}
        </a>
      </li>

      <li class="line"></li>
      <li v-if="!isMobile">
        <a @click="resetMenuWidth" :class="{ 'pointer-events-none opacity-40': !canResetMenuWidth }">
          <mdi:arrow-collapse-horizontal />
          {{ $t("toolbar.reset-sidebar-width") }}
        </a>
      </li>
      <li>
        <router-link v-if="!settingsAsPopup" :to="{ name: '/settings' }">
          <mdi:cog />
          {{ $t("title.settings") }}
        </router-link>
        <a v-else @click="openSettings">
          <mdi:cog />
          {{ $t("title.settings") }}
        </a>
      </li>

      <!-- Container Actions (Enabled via config) -->
      <template v-if="enableActions && !historical">
        <li class="line"></li>
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
        <li>
          <button @click="update()" :disabled="actionStates.update">
            <carbon:upgrade />
            {{ container.isSwarm ? $t("toolbar.update-service") : $t("toolbar.update") }}
          </button>
        </li>
      </template>

      <template v-if="enableShell && !historical">
        <li class="line"></li>
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
    </ul>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import { allLevels } from "@/composable/logContext";
import LogAnalytics from "../LogViewer/LogAnalytics.vue";
import Terminal from "@/components/Terminal.vue";

const { showSearch } = useSearchFilter();
const { actionsMenuOpen, canHover, focusOnTap, hideMenu, onLeave } = useActionsMenu();
const { openSettings } = useSettingsModal();
const { enableActions, enableShell, enableDownload } = config;
const { streamConfig, hasComplexLogs, levels } = useLoggingContext();
const showDrawer = useDrawer();

const { container, historical = false } = defineProps<{ container: Container; historical?: boolean }>();
const clear = defineEmit();
const { actionStates, start, stop, restart, update } = useContainerActions(toRef(() => container));

const router = useRouter();
const { copy, copied, isSupported } = useClipboard({ legacy: true });
const { t } = useI18n();
const { showToast } = useToast();

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

const containerRef = computed(() => [container]);
const { downloadUrl, isFiltered } = useDownloadUrl(
  containerRef,
  streamConfig,
  levels,
  toRef(() => container.name),
);

const disableRestart = computed(() => actionStates.stop || actionStates.start || actionStates.restart);

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

li.line {
  @apply bg-base-content/20 h-px;
}

a {
  @apply whitespace-nowrap;
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
