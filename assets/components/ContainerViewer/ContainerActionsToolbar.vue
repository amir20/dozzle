<template>
  <div class="dropdown dropdown-end dropdown-hover">
    <label tabindex="0" class="btn btn-ghost btn-sm w-10 gap-0.5 px-2">
      <carbon:circle-solid class="text-red w-2.5" v-if="streamConfig.stderr" />
      <carbon:circle-solid class="text-blue w-2.5" v-if="streamConfig.stdout" />
    </label>
    <ul
      tabindex="0"
      class="menu dropdown-content rounded-box bg-base-200 border-base-content/20 z-50 w-52 border p-1 shadow-sm"
      @click="hideMenu"
    >
      <li v-if="!historical">
        <a @click.prevent="clear()">
          <octicon:trash-24 /> {{ $t("toolbar.clear") }}
          <KeyShortcut char="k" :modifiers="['shift', 'meta']" />
        </a>
      </li>
      <li>
        <a :href="downloadUrl" download> <octicon:download-24 /> {{ $t("toolbar.download") }} </a>
      </li>
      <li v-if="!historical">
        <a @click.prevent="showSearch = true">
          <mdi:magnify /> {{ $t("toolbar.search") }}
          <KeyShortcut char="f" />
        </a>
      </li>
      <li v-if="hasComplexLogs">
        <a @click.prevent="showDrawer(LogAnalytics, { container }, 'lg')">
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
      </template>

      <template v-if="enableShell && !historical">
        <li class="line"></li>
        <li>
          <a @click.prevent="showDrawer(Terminal, { container, action: 'attach' }, 'lg')">
            <ri:terminal-window-fill />
            {{ $t("toolbar.attach") }}
            <KeyShortcut char="a" :modifiers="['shift', 'meta']" />
          </a>
        </li>
        <li>
          <a @click.prevent="showDrawer(Terminal, { container, action: 'exec' }, 'lg')">
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
const { enableActions, enableShell } = config;
const { streamConfig, hasComplexLogs, levels } = useLoggingContext();
const showDrawer = useDrawer();

const { container, historical = false } = defineProps<{ container: Container; historical?: boolean }>();
const clear = defineEmit();
const { actionStates, start, stop, restart } = useContainerActions(toRef(() => container));

onKeyStroke("f", (e) => {
  if (hasComplexLogs.value) {
    if ((e.ctrlKey || e.metaKey) && e.shiftKey) {
      showDrawer(LogAnalytics, { container }, "lg");
      e.preventDefault();
    }
  }
});
if (enableShell) {
  onKeyStroke("a", (e) => {
    if ((e.ctrlKey || e.metaKey) && e.shiftKey) {
      showDrawer(Terminal, { container, action: "attach" }, "lg");
      e.preventDefault();
    }
  });

  onKeyStroke("e", (e) => {
    if ((e.ctrlKey || e.metaKey) && e.shiftKey) {
      showDrawer(Terminal, { container, action: "exec" }, "lg");
      e.preventDefault();
    }
  });
}

const downloadParams = computed(() =>
  Object.entries(toValue(streamConfig))
    .filter(([, value]) => value)
    .reduce((acc, [key]) => ({ ...acc, [key]: "1" }), {}),
);

const downloadUrl = computed(() =>
  withBase(
    `/api/containers/${container.host}~${container.id}/download?${new URLSearchParams(downloadParams.value).toString()}`,
  ),
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

li.line {
  @apply bg-base-content/20 h-px;
}

a {
  @apply whitespace-nowrap;
}

.menu li ul {
  margin-inline-start: 0;
  &:before {
    display: none;
  }
}
</style>
