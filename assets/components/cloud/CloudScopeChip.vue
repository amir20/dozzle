<template>
  <!-- Two segments: the containers on screen, or every container. Only on a page
       about containers; anywhere else the panels already show the instance. -->
  <div
    v-if="inView.length"
    role="group"
    class="border-base-content/15 bg-base-200/40 max-w-full rounded-full border p-0.5 text-xs"
    :class="isMobile ? 'flex' : 'inline-flex'"
    :aria-label="$t('cloud-rail.scope')"
  >
    <button
      v-for="segment in segments"
      :key="segment.id"
      type="button"
      class="flex min-w-0 items-center justify-center gap-1.5 rounded-full px-3 py-1 transition-colors"
      :class="[
        segment.active ? 'bg-base-300 font-semibold' : 'text-base-content/60 hover:bg-base-300',
        { 'flex-1': isMobile },
      ]"
      :aria-pressed="segment.active"
      @click="segment.select()"
    >
      <span class="truncate" :class="{ 'font-mono': segment.mono }">{{ segment.label }}</span>
      <span v-if="segment.count !== undefined" class="text-base-content/40 font-mono">{{ segment.count }}</span>
    </button>
  </div>
</template>

<script lang="ts" setup>
const { inView, scoped, total, widen, narrow } = useCloudScope();
const { panel } = useCloudRail();
const { hosts } = useHosts();
const { t } = useI18n();

// Widened, Metrics is about the hosts rather than a list of containers, so its
// wide side says so and counts hosts.
const hostWide = computed(() => panel.value === "metrics");
const hostCount = computed(() => Object.values(hosts.value).filter((h) => h.available).length);

const segments = computed(() => [
  {
    id: "view",
    // A merged view or a stack has no one name, so it is labelled by count.
    label: inView.value.length === 1 ? inView.value[0].name : t("cloud-rail.scope-n", { n: inView.value.length }),
    mono: inView.value.length === 1,
    count: undefined,
    active: scoped.value,
    select: narrow,
  },
  {
    id: "all",
    label: hostWide.value ? t("cloud-rail.scope-all-hosts") : t("cloud-rail.scope-all"),
    mono: false,
    count: hostWide.value ? hostCount.value : total.value,
    active: !scoped.value,
    select: widen,
  },
]);
</script>
