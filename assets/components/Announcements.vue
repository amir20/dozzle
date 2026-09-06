<template>
  <Dropdown
    class="dropdown-end"
    @click="config.releaseCheckMode === 'manual' && fetchReleases()"
    @closed="releaseSeen = mostRecent?.tag ?? config.version"
  >
    <template #trigger>
      <mdi:announcement class="icon-wiggle size-6 -rotate-12" />
      <template v-if="announcements.length > 0 && releaseSeen != mostRecent?.tag">
        <span class="bg-red absolute top-0 right-px size-2 animate-ping rounded-full opacity-75"></span>
        <span class="bg-red absolute top-0 right-px size-2 rounded-full"></span>
      </template>
    </template>
    <template #content>
      <div class="flex max-h-[70vh] w-80 flex-col">
        <!-- The API stops walking releases at the running version, so everything below is an
             update you don't have yet. Nothing said so before. -->
        <div
          v-if="announcements.length > 0"
          class="border-base-content/15 flex items-baseline justify-between gap-2 border-b px-2 pb-2"
        >
          <span class="text-sm font-semibold">{{ $t("releases.newer", announcements.length) }}</span>
          <span class="text-base-content/60 font-mono text-xs">{{ config.version }}</span>
        </div>

        <ul class="min-h-0 flex-1 space-y-4 overflow-y-auto p-2">
          <li v-if="announcements.length === 0">
            <div class="text-base-content/80 text-sm">
              {{ $t("releases.no_releases") }}
            </div>
          </li>
          <li v-for="release in announcements" :key="release.tag">
            <template v-if="release.announcement">
              <div class="flex items-baseline gap-1">
                <carbon:information class="text-info self-center" />
                <a
                  :href="release.htmlUrl"
                  class="link-primary text-lg font-bold"
                  target="_blank"
                  rel="noreferrer noopener"
                >
                  {{ release.name }}
                </a>
                <span class="ml-1 text-xs"><RelativeTime :date="release.createdAt" /></span>
              </div>
              <div class="text-base-content/80 text-sm">
                {{ release.body }}
              </div>
            </template>
            <template v-else>
              <div class="flex items-baseline gap-1">
                <carbon:warning class="stroke-orange self-center" v-if="release.breaking > 0" />
                <a
                  :href="release.htmlUrl"
                  class="link-primary text-lg font-bold"
                  target="_blank"
                  rel="noreferrer noopener"
                >
                  {{ release.name }}
                </a>
                <span class="ml-1 text-xs"><RelativeTime :date="release.createdAt" /></span>
                <!-- Red read as a warning on what is the good news in this list. -->
                <Tag class="bg-primary! text-primary-content ml-auto px-1 py-1 text-xs" v-if="release.latest">
                  {{ $t("releases.latest") }}
                </Tag>
              </div>
              <div class="text-base-content/80 text-sm">
                {{ summary(release) }}
              </div>
            </template>
          </li>
        </ul>

        <div v-if="announcements.length > 0" class="border-base-content/15 border-t px-2 pt-2">
          <a
            href="https://github.com/amir20/dozzle/releases"
            class="link-primary text-xs"
            target="_blank"
            rel="noreferrer noopener"
          >
            {{ $t("releases.all") }}
          </a>
        </div>
      </div>
    </template>
  </Dropdown>
</template>

<script setup lang="ts">
import { useAnnouncements } from "@/stores/announcements";

const { announcements, mostRecent, fetchReleases } = useAnnouncements();
const { t } = useI18n();

const releaseSeen = useProfileStorage("releaseSeen", config.version);

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
