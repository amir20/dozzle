<template>
  <!-- App logo carrying its status as a corner badge. Two glyphs side by side turned a
       dense container list into noise, and the logo is the part people scan for, so status
       rides on top of it instead of competing for its own column. Sizing comes from the
       parent; everything inside is relative so one component works at any scale. -->
  <div class="container-icon relative inline-flex shrink-0" :data-status="status" :title="status">
    <template v-if="src">
      <img
        :src="src"
        :alt="slug"
        class="size-full object-contain"
        :class="{ 'opacity-40 grayscale': !isRunning }"
        loading="lazy"
        decoding="async"
      />
      <cil:check-circle v-if="health === 'healthy'" class="status-badge" />
      <mdi:alert-circle v-else-if="health === 'unhealthy'" class="status-badge" />
      <cil:media-pause v-else-if="state === 'paused'" class="status-badge" />
      <span v-else class="status-badge status-dot"></span>
    </template>

    <!-- Nothing matched the image, so fall back to the plain status glyph. -->
    <template v-else>
      <cil:check-circle v-if="health === 'healthy'" class="size-full" />
      <mdi:alert-circle v-else-if="health === 'unhealthy'" class="size-full" />
      <cil:media-pause v-else-if="state === 'paused'" class="size-full" />
      <mdi:circle-medium v-else class="size-full" />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { ContainerHealth, ContainerState } from "@/types/Container";
import { showAppIcons } from "@/stores/settings";

const { state, health, slug } = defineProps<{
  state: ContainerState;
  health?: ContainerHealth;
  slug?: string;
}>();

const { isDark } = useResolvedTheme();

const status = computed(() => health ?? state);
const isRunning = computed(() => state === "running" || state === "restarting");
const src = computed(() => (showAppIcons.value ? iconUrl(slug, isDark.value) : undefined));
</script>

<style scoped>
@reference "@/main.css";

/* Overhangs the logo's top-right corner. The ring punches a hole in whatever the logo
   puts underneath so the badge stays legible on busy artwork. */
.status-badge {
  position: absolute;
  top: -18%;
  right: -18%;
  width: 58%;
  height: 58%;
  min-width: 9px;
  min-height: 9px;
  border-radius: 9999px;
  @apply bg-base-100 ring-base-100 ring-2;
}

/* No glyph of its own, so it paints in the status colour rather than inheriting it. */
.status-dot {
  top: -10%;
  right: -10%;
  width: 40%;
  height: 40%;
  min-width: 7px;
  min-height: 7px;
  background-color: currentColor;
}

[data-status="unhealthy"],
[data-status="exited"],
[data-status="dead"] {
  @apply text-red;
}

[data-status="healthy"],
[data-status="running"] {
  @apply text-green;
}

[data-status="starting"],
[data-status="restarting"],
[data-status="paused"] {
  @apply text-orange;
}

[data-status="created"],
[data-status="deleted"] {
  @apply text-base-content/40;
}
</style>
