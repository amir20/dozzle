<template>
  <Dropdown placement="bottom-end" @opened="opened" @closed="closed">
    <template #trigger>
      <mdi:announcement class="icon-wiggle size-6 -rotate-12" />
      <template v-if="unseenCount > 0">
        <span
          class="bg-red absolute -top-1 -right-1.5 h-4 min-w-4 animate-ping rounded-full opacity-75 motion-reduce:animate-none"
        ></span>
        <span
          class="bg-red absolute -top-1 -right-1.5 flex h-4 min-w-4 items-center justify-center rounded-full px-1 font-mono text-[0.625rem] leading-none font-bold text-white"
        >
          {{ unseenCount > 9 ? "9+" : unseenCount }}
        </span>
      </template>
    </template>
    <template #content>
      <div class="flex max-h-[70vh] w-80 flex-col">
        <!-- One scroll area for both groups, so the footer stays put and nothing nests a scroll. -->
        <div class="min-h-0 flex-1 overflow-y-auto">
          <!-- Updates Dozzle made lately. Kept in memory on the server, so a restart empties it.
               Past a few, one row sums them up and opens on click. -->
          <template v-if="recentUpdates.length > 0">
            <button
              v-if="recentUpdates.length > SUMMARIZE_PAST"
              class="hover:bg-base-300 flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left transition-colors"
              :aria-expanded="showAllUpdates"
              @click="showAllUpdates = !showAllUpdates"
            >
              <span
                class="flex size-6 shrink-0 items-center justify-center rounded-full"
                :class="anyUndone ? 'bg-warning/10 text-warning' : 'bg-info/10 text-info'"
              >
                <mdi:update class="size-3.5" />
              </span>
              <span class="min-w-0 flex-1">
                <span class="block truncate text-sm font-semibold">
                  {{ $t("releases.updates_summary", updatedContainers.length) }}
                </span>
                <span class="text-base-content/60 block truncate text-xs">{{ updatedNames }}</span>
              </span>
              <mdi:chevron-down
                class="size-4 shrink-0 opacity-40 transition-transform"
                :class="{ 'rotate-180': showAllUpdates }"
              />
            </button>
            <ul v-if="recentUpdates.length <= SUMMARIZE_PAST || showAllUpdates">
              <li v-for="update in recentUpdates" :key="`${update.host}/${update.newId}/${update.at}`">
                <component
                  :is="allContainersById[update.newId] ? RouterLink : 'div'"
                  :to="{ name: '/container/[id]', params: { id: update.newId } }"
                  class="flex items-center gap-2 rounded-md px-2 py-1.5"
                  :class="{ 'hover:bg-base-300 transition-colors': allContainersById[update.newId] }"
                  @click="trackUsage('megaphone.update')"
                >
                  <span
                    class="flex size-6 shrink-0 items-center justify-center rounded-full"
                    :class="updateKind(update) === 'undone' ? 'bg-warning/10 text-warning' : 'bg-info/10 text-info'"
                  >
                    <mdi:update v-if="updateKind(update) === 'updated'" class="size-3.5" />
                    <mdi:restore v-else class="size-3.5" />
                  </span>
                  <span class="min-w-0 flex-1">
                    <!-- The icon already says "updated"; only an undone or rolled-back one is named. -->
                    <span class="flex items-baseline gap-2">
                      <span class="truncate text-sm font-semibold">{{ update.name }}</span>
                      <span v-if="updateKind(update) !== 'updated'" class="text-base-content/60 shrink-0 text-xs">
                        {{ $t(`update-marker.${updateKind(update)}`) }}
                      </span>
                    </span>
                    <!-- Only the image it moved to: both sides do not fit at this width, and the
                         log view's marker shows the pair. -->
                    <span class="text-base-content/60 block truncate font-mono text-xs">
                      {{ updateLabels(update).to }}
                    </span>
                  </span>
                  <RelativeTime :date="new Date(update.at)" class="text-base-content/40 shrink-0 text-xs" />
                </component>
              </li>
            </ul>
            <div class="bg-base-content/10 m-2 h-px"></div>
          </template>

          <!-- The last scheduled clean up, when it freed something or failed somewhere. -->
          <template v-if="lastAutoPrune">
            <div class="flex items-center gap-2 px-2 py-1.5">
              <span
                class="flex size-6 shrink-0 items-center justify-center rounded-full"
                :class="lastAutoPrune.failed ? 'bg-warning/10 text-warning' : 'bg-success/10 text-success'"
              >
                <mdi:broom class="size-3.5" />
              </span>
              <span class="min-w-0 flex-1">
                <span class="block truncate text-sm font-semibold">
                  {{ $t("reclaim.freed", { size: formatBytes(lastAutoPrune.spaceReclaimed, { decimals: 1 }) }) }}
                </span>
                <span class="text-base-content/60 block truncate text-xs">
                  {{
                    lastAutoPrune.failed
                      ? $t("reclaim.last-run-failed", { count: lastAutoPrune.failed }, lastAutoPrune.failed)
                      : $t("reclaim.scheduled-run")
                  }}
                </span>
              </span>
              <RelativeTime :date="new Date(lastAutoPrune.at)" class="text-base-content/40 shrink-0 text-xs" />
            </div>
            <div class="bg-base-content/10 m-2 h-px"></div>
          </template>

          <!-- The API stops walking releases at the running version, so every one listed is a
               release you don't have yet. -->
          <div v-if="announcements.length === 0" class="flex items-center gap-2 px-2 py-1.5">
            <span class="bg-success/10 text-success flex size-6 shrink-0 items-center justify-center rounded-full">
              <mdi:check class="size-3.5" />
            </span>
            <span class="text-sm">{{ $t("releases.no_releases") }}</span>
          </div>
          <template v-else>
            <button
              v-if="announcements.length > SUMMARIZE_PAST"
              class="hover:bg-base-300 flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left transition-colors"
              :aria-expanded="showAllReleases"
              @click="showAllReleases = !showAllReleases"
            >
              <span
                class="flex size-6 shrink-0 items-center justify-center rounded-full"
                :class="releaseTotals.breaking > 0 ? 'bg-warning/10 text-warning' : 'bg-info/10 text-info'"
              >
                <mdi:alert-outline v-if="releaseTotals.breaking > 0" class="size-3.5" />
                <mdi:tag-outline v-else class="size-3.5" />
              </span>
              <span class="min-w-0 flex-1">
                <span class="block truncate text-sm font-semibold">
                  {{ $t("releases.newer", announcements.length) }}
                </span>
                <span class="text-base-content/60 block truncate text-xs">{{ summary(releaseTotals) }}</span>
              </span>
              <mdi:chevron-down
                class="size-4 shrink-0 opacity-40 transition-transform"
                :class="{ 'rotate-180': showAllReleases }"
              />
            </button>
            <ul v-if="announcements.length <= SUMMARIZE_PAST || showAllReleases">
              <li v-for="release in announcements" :key="release.tag">
                <a
                  :href="release.htmlUrl"
                  target="_blank"
                  rel="noreferrer noopener"
                  class="hover:bg-base-300 flex items-center gap-2 rounded-md px-2 py-1.5 transition-colors"
                  @click="trackUsage('megaphone.release')"
                >
                  <span
                    class="flex size-6 shrink-0 items-center justify-center rounded-full"
                    :class="
                      !release.announcement && release.breaking > 0
                        ? 'bg-warning/10 text-warning'
                        : 'bg-info/10 text-info'
                    "
                  >
                    <mdi:information-outline v-if="release.announcement" class="size-3.5" />
                    <mdi:alert-outline v-else-if="release.breaking > 0" class="size-3.5" />
                    <mdi:tag-outline v-else class="size-3.5" />
                  </span>
                  <span class="min-w-0 flex-1">
                    <span class="flex items-baseline gap-2">
                      <span class="truncate font-mono text-sm font-semibold">{{ release.name }}</span>
                      <span v-if="release.latest" class="status-pill status-pill-success shrink-0">
                        {{ $t("releases.latest") }}
                      </span>
                    </span>
                    <span class="text-base-content/60 block truncate text-xs">
                      {{ release.announcement ? release.body : summary(release) }}
                    </span>
                  </span>
                  <RelativeTime :date="release.createdAt" class="text-base-content/40 shrink-0 text-xs" />
                </a>
              </li>
            </ul>
          </template>
        </div>

        <div class="bg-base-content/10 m-2 h-px"></div>
        <a
          href="https://github.com/amir20/dozzle/releases"
          target="_blank"
          rel="noreferrer noopener"
          class="hover:bg-base-300 flex items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors"
          @click="trackUsage('megaphone.release')"
        >
          <mdi:github class="size-4 opacity-60" />
          <span class="flex-1">{{ $t("releases.all") }}</span>
          <span class="text-base-content/40 font-mono text-xs">{{ config.version }}</span>
          <mdi:open-in-new class="size-3.5 opacity-40" />
        </a>
      </div>
    </template>
  </Dropdown>
</template>

<script setup lang="ts">
import { RouterLink } from "vue-router";
import { useAnnouncements } from "@/stores/announcements";
import { updateKind, updateLabels } from "@/models/ContainerUpdate";

const { announcements, mostRecent, fetchReleases } = useAnnouncements();
const { recentUpdates, lastAutoPrune, allContainersById } = storeToRefs(useContainerStore());
const { t } = useI18n();

const releaseSeen = useProfileStorage("releaseSeen", config.version);
// When the newest update or clean up the dropdown showed finished, so only ones after it
// light the dot. Kept per browser rather than in the profile: both only live until a restart.
const updatesSeen = useStorage("DOZZLE_UPDATES_SEEN", "");

const newestUpdateAt = computed(() => {
  const times = [recentUpdates.value[0]?.at, lastAutoPrune.value?.at].filter((at): at is string => !!at);
  return times.sort((a, b) => Date.parse(b) - Date.parse(a))[0] ?? "";
});

// Announcements are newest first, so the unseen releases are the ones above the last one
// seen. A seen tag no longer listed (the running version, or one since dropped) means
// every listed release is new.
const unseenReleases = computed(() => {
  const seen = announcements.value.findIndex((r) => r.tag === releaseSeen.value);
  return seen === -1 ? announcements.value.length : seen;
});
const unseenUpdates = computed(() => {
  const seen = updatesSeen.value ? Date.parse(updatesSeen.value) : 0;
  const prune = lastAutoPrune.value && Date.parse(lastAutoPrune.value.at) > seen ? 1 : 0;
  return recentUpdates.value.filter((u) => Date.parse(u.at) > seen).length + prune;
});
const unseenCount = computed(() => unseenReleases.value + unseenUpdates.value);

// Past this many, a group is one summary row that opens on click.
const SUMMARIZE_PAST = 3;
const showAllUpdates = ref(false);
const showAllReleases = ref(false);

const anyUndone = computed(() => recentUpdates.value.some((u) => updateKind(u) === "undone"));
// A container updated twice is one name, so the summary counts names, not updates.
const updatedContainers = computed(() => [...new Set(recentUpdates.value.map((u) => u.name))]);
// The first few names, newest first, and how many more: "web, db, api +9".
const updatedNames = computed(() => {
  const names = updatedContainers.value;
  const shown = names.slice(0, SUMMARIZE_PAST).join(", ");
  return names.length > SUMMARIZE_PAST ? `${shown} +${names.length - SUMMARIZE_PAST}` : shown;
});
const releaseTotals = computed(() =>
  announcements.value
    .filter((r) => !r.announcement)
    .reduce(
      (acc, r) => ({
        features: acc.features + r.features,
        bugFixes: acc.bugFixes + r.bugFixes,
        breaking: acc.breaking + r.breaking,
      }),
      { features: 0, bugFixes: 0, breaking: 0 },
    ),
);

function opened() {
  trackUsage("megaphone.open");
  if (config.releaseCheckMode === "manual") fetchReleases();
}

function closed() {
  releaseSeen.value = mostRecent.value?.tag ?? config.version;
  if (newestUpdateAt.value) updatesSeen.value = newestUpdateAt.value;
  showAllUpdates.value = false;
  showAllReleases.value = false;
}

function summary(release: { features: number; bugFixes: number; breaking: number }) {
  if (release.features > 0 && release.bugFixes > 0 && release.breaking > 0) {
    return t("releases.three_parts", {
      first: t("releases.breaking", { count: release.breaking }),
      second: t("releases.features", { count: release.features }),
      third: t("releases.bugFixes", { count: release.bugFixes }),
    });
  }

  if (release.features > 0 && release.bugFixes > 0) {
    return t("releases.two_parts", {
      first: t("releases.features", { count: release.features }),
      second: t("releases.bugFixes", { count: release.bugFixes }),
    });
  }

  if (release.features > 0 && release.breaking > 0) {
    return t("releases.two_parts", {
      first: t("releases.features", { count: release.features }),
      second: t("releases.breaking", { count: release.breaking }),
    });
  }

  if (release.bugFixes > 0 && release.breaking > 0) {
    return t("releases.two_parts", {
      first: t("releases.bugFixes", { count: release.bugFixes }),
      second: t("releases.breaking", { count: release.breaking }),
    });
  }

  if (release.features > 0) {
    return t("releases.features", { count: release.features });
  }

  if (release.bugFixes > 0) {
    return t("releases.bugFixes", { count: release.bugFixes });
  }

  if (release.breaking > 0) {
    return t("releases.breaking", { count: release.breaking });
  }
}
</script>

<style scoped></style>
