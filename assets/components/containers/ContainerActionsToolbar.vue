<template>
  <Popover hover sheet placement="bottom-end" panel-class="log-menu-panel">
    <template #trigger>
      <button type="button" class="icon-btn btn btn-ghost btn-sm relative w-8 px-0" data-testid="log-actions">
        <mdi:dots-vertical class="size-4" />
        <span
          v-if="showImageUpdateAlert"
          class="absolute end-0.5 top-0.5 flex size-1.5"
          :title="$t('toolbar.update-available')"
        >
          <span class="bg-warning absolute size-full rounded-full opacity-75 motion-safe:animate-ping"></span>
          <span class="bg-warning relative size-full rounded-full"></span>
        </span>
        <!-- On a phone the menu is the only door to the alerts panel, so the dot
             that door carries shows on the closed menu too. -->
        <span
          v-else-if="cloudLinked && isMobile && unseenAlerts"
          class="bg-warning absolute end-0.5 top-0.5 size-1.5 rounded-full"
        ></span>
      </button>
    </template>
    <ul class="menu log-menu w-full p-0">
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
          <KeyShortcut v-if="isMainView" char="p" :modifiers="[]" />
        </a>
      </li>
      <li v-if="!historical">
        <a @click="clear()">
          <octicon:trash-24 /> {{ $t("toolbar.clear") }}
          <KeyShortcut char="l" :modifiers="['shift', 'meta']" />
        </a>
      </li>
      <li v-if="hasComplexLogs">
        <a @click="openAnalytics()">
          <ph:file-sql /> {{ $t("analytics.title") }}
          <KeyShortcut v-if="isMainView" char="s" :modifiers="[]" />
        </a>
      </li>
      <!-- A phone has no rail strip, so this one row is the way into all three
           panels: named for Cloud rather than chat, carrying the alerts dot the
           strip would have shown, and opening where the reader needs to look. -->
      <li v-if="cloudLinked && isMobile">
        <a @click="openCloud(unseenAlerts)">
          <mdi:cloud-outline /> {{ $t("cloud.title") }}
          <span v-if="unseenAlerts" class="bg-warning ms-auto size-1.5 rounded-full"></span>
        </a>
      </li>
      <li v-else-if="cloudLinked">
        <a @click="openRail('chat')">
          <mdi:message-outline /> {{ $t("cloud-chat.title") }}
          <KeyShortcut v-if="isMainView" char="." :modifiers="[]" />
        </a>
      </li>
      <!-- The filter chip in the title bar carries these where it fits; a phone and a
           pinned column have no room for it, so they land here instead. Time is phone
           only: a range is a route, and a pinned column picking one would move the
           main view instead. -->
      <template v-if="filters">
        <li class="section">{{ $t("toolbar.section-filters") }}</li>
        <LogTimeRows v-if="!historical && isMobile" nested :container :range="timeRange" />
        <LogFilterRows />
      </template>

      <LogExportRows :containers="[container]" :name="container.name" :shortcuts="isMainView" />
      <li>
        <a @click="copyPermalink()">
          <material-symbols:link />
          {{ $t("toolbar.copy-permalink") }}
        </a>
      </li>

      <!-- Container Actions (Enabled via config) -->
      <li class="section" v-if="!historical">{{ $t("toolbar.section-container") }}</li>
      <li v-if="!historical">
        <a @click="showDrawer(ContainerInspect, { container })">
          <mdi:information-outline /> {{ $t("toolbar.inspect") }}
          <KeyShortcut v-if="isMainView" char="i" :modifiers="[]" />
        </a>
      </li>
      <template v-if="enableActions && !historical">
        <!-- Kubernetes has no stop or start for one container, only restart. -->
        <li v-if="canStartStop && power">
          <button @click="stop()" :disabled="actionStates.stop || actionStates.restart" v-if="power === 'stop'">
            <carbon:stop-filled-alt /> {{ $t("toolbar.stop") }}
          </button>

          <button @click="start()" :disabled="actionStates.start || actionStates.restart" v-else>
            <carbon:play /> {{ $t("toolbar.start") }}
          </button>
        </li>
        <li v-if="power">
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
        <!-- Restart above replaces this one pod; this rolls every pod of the workload. -->
        <li v-if="workload">
          <button @click="rolloutRestart(workload)" :disabled="rollingOut" :title="`${workload.kind}/${workload.name}`">
            <carbon:renew :class="{ 'animate-spin': rollingOut, 'text-secondary': rollingOut }" />
            {{ $t("toolbar.rollout-restart", { kind: workload.kind }) }}
          </button>
        </li>
        <li v-if="imageUpdatable">
          <button @click="update({ self: isSelfContainer })" :disabled="actionStates.update">
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
            <KeyShortcut v-if="isMainView" char="a" :modifiers="[]" />
          </a>
        </li>
        <li>
          <a @click="showDrawer(Terminal, { container, action: 'exec' }, 'lg')">
            <material-symbols:terminal />
            {{ $t("toolbar.shell") }}
            <KeyShortcut v-if="isMainView" char="e" :modifiers="[]" />
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
import { Container, powerAction } from "@/models/Container";
// Drawer-only, and it drags in apache-arrow; loading it eagerly put both on every log view.
const LogAnalytics = defineAsyncComponent(() => import("@/components/logs/LogAnalytics.vue"));
import Terminal from "./Terminal.vue";
import ContainerInspect from "./ContainerInspect.vue";

const { showSearch } = useSearchFilter();
const { linked: cloudLinked } = useCloudSurface();
const { openRail, openCloud } = useCloudRail();
const { unseen: unseenAlerts } = useViewAlerts();
const { enableActions, enableShell } = config;
const { hasComplexLogs, timeRange } = useLoggingContext();
const showDrawer = useDrawer();

const {
  container,
  historical = false,
  filters = false,
} = defineProps<{
  container: Container;
  historical?: boolean;
  /** Carry the time, stream and level rows, for where the filter chip is not shown. */
  filters?: boolean;
}>();
const clear = defineEmit();
const { actionStates, start, stop, restart, update } = useContainerActions(toRef(() => container));
const canStartStop = config.mode !== "k8s";
const workload = computed(() => containerWorkload(container));
const { restarting: rollingOut, rolloutRestart } = useRolloutRestart();
const {
  showAlert: showImageUpdateAlert,
  isSelf: isSelfContainer,
  updatable: imageUpdatable,
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
const { copy } = useCopy();

async function copyPermalink() {
  const url = router.resolve({
    name: "/show",
    query: { name: container.name, host: container.host },
  }).href;

  await copy(new URL(url, window.location.origin).href);
}

async function copyImageReference() {
  await copy(container.image, { message: escapeHtml(container.image) });
}

// A pinned column mounts its own toolbar; only the page's acts on a key, or one press
// would open a drawer per column.
const isMainView = isViewContextOwner();
const canShell = isMainView && enableShell && !historical;
const openAnalytics = () => showDrawer(LogAnalytics, { containers: [container] }, "lg");
const openTerminal = (action: "attach" | "exec") => showDrawer(Terminal, { container, action }, "lg");

useLetterShortcut("s", openAnalytics, () => isMainView && hasComplexLogs.value);
useLetterShortcut("i", () => showDrawer(ContainerInspect, { container }), isMainView && !historical);
useLetterShortcut("p", () => (pinned.value = !pinned.value), isMainView);
useLetterShortcut("a", () => openTerminal("attach"), canShell);
useLetterShortcut("e", () => openTerminal("exec"), canShell);

// The chords the letters replaced, kept working for one release.
const legacyChord = (enabled: () => boolean, run: () => void) => (e: KeyboardEvent) => {
  if ((e.ctrlKey || e.metaKey) && e.shiftKey && enabled()) {
    run();
    e.preventDefault();
  }
};
onKeyStroke(
  ["f", "F"],
  legacyChord(() => isMainView && hasComplexLogs.value, openAnalytics),
);
onKeyStroke(
  ["a", "A"],
  legacyChord(
    () => canShell,
    () => openTerminal("attach"),
  ),
);
onKeyStroke(
  ["e", "E"],
  legacyChord(
    () => canShell,
    () => openTerminal("exec"),
  ),
);

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

// Nothing to act on once the container is deleted, and Docker will not start a paused one.
const power = computed(() => powerAction(container));
const disableRestart = computed(() => actionStates.stop || actionStates.start || actionStates.restart);
</script>
