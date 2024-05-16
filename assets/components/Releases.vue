<template>
  <ul class="space-y-4 p-2">
    <li v-for="release in releases" v-if="releases?.length">
      <div class="flex items-baseline gap-1">
        <carbon:warning class="self-center stroke-orange" v-if="release.breaking > 0" />
        <a :href="release.htmlUrl" class="link-primary text-lg font-bold" target="_blank" rel="noreferrer noopener">
          {{ release.name }}
        </a>
        <span class="ml-1 text-xs"><distance-time :date="new Date(release.createdAt)" /></span>
        <Tag class="ml-auto bg-red px-1 py-1 text-xs" v-if="release.tag === latest?.tag">
          {{ $t("releases.latest") }}
        </Tag>
      </div>
      <div class="text-sm text-base-content/80">
        {{ summary(release) }}
      </div>
    </li>
    <li v-else>
      <div class="text-sm text-base-content/80">
        {{ $t("releases.no_releases") }}
      </div>
    </li>
  </ul>
</template>

<script setup lang="ts">
const { releases, latest } = useReleases();
const { t } = useI18n();

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
