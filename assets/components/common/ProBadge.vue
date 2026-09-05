<template>
  <a
    v-if="isPro"
    :href="cloudUrl"
    target="_blank"
    rel="noreferrer noopener"
    class="pro-badge status-pill status-pill-secondary"
    :title="$t('cloud.pro-badge')"
    data-testid="pro-badge"
    @click.stop
  >
    PRO
  </a>
</template>

<script lang="ts" setup>
const cloudUrl = __CLOUD_URL__;
const { isPro, cloudConfig, ensureCloudStatus } = useCloudConfig();

// The badge can mount before the shared cloud config resolves, so ask again
// once it lands. `ensureCloudStatus` is a no-op when the status is already in.
watch(
  () => cloudConfig.value?.linked,
  () => ensureCloudStatus(),
  { immediate: true },
);
</script>

<style scoped>
@reference "@/main.css";

/* Rides on the app's status-pill idiom (mono, uppercase, tracked, thin border)
 * in the logo's gold, and only trims it down to sit beside the wordmark. The
 * right padding drops the tracking added after the final letter so the text
 * stays optically centered. */
.pro-badge {
  @apply border-secondary/50 relative shrink-0 overflow-hidden rounded-[3px] py-px pl-[0.3rem] text-[0.6875rem] leading-[1.25] font-bold;
  padding-right: calc(0.3rem - 0.1em);
  transition:
    border-color 150ms ease-out,
    background-color 150ms ease-out;
}

.pro-badge:hover {
  @apply border-secondary/80 bg-secondary/20;
}

/* A single narrow glint, in the accent color rather than white so it stays
 * crisp on both themes. Long pause between passes keeps it a sheen. */
.pro-badge::after {
  content: "";
  @apply pointer-events-none absolute inset-0;
  background: linear-gradient(
    105deg,
    transparent 44%,
    color-mix(in oklab, var(--color-secondary) 55%, transparent) 50%,
    transparent 56%
  );
  transform: translateX(-160%);
  animation: pro-shine 5s ease-in-out infinite;
}

@keyframes pro-shine {
  0%,
  78% {
    transform: translateX(-160%);
  }
  100% {
    transform: translateX(160%);
  }
}

@media (prefers-reduced-motion: reduce) {
  .pro-badge::after {
    animation: none;
  }
}
</style>
