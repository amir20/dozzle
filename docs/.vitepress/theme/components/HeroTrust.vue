<script lang="ts" setup>
import { computed } from "vue";
import { data } from "../../activity.data";

const cues = [
  { icon: "mdi:scale-balance", label: "MIT licensed" },
  { icon: "mdi:package-variant-closed", label: "Single binary, no database" },
  { icon: "mdi:server-security", label: "Self-hosted, your logs stay on your network" },
];

const released = computed(() => {
  if (!data.release) return null;
  const days = Math.floor((Date.now() - new Date(data.release.date).getTime()) / 86_400_000);
  if (days <= 0) return "released today";
  if (days === 1) return "released yesterday";
  if (days < 30) return `released ${days} days ago`;
  const months = Math.round(days / 30);
  return `released ${months} month${months === 1 ? "" : "s"} ago`;
});
</script>

<template>
  <div class="mt-8 flex flex-col gap-2 text-sm text-(--vp-c-text-2)">
    <ul class="m-0! flex list-none flex-wrap gap-x-5 gap-y-2 p-0!">
      <li v-for="cue in cues" :key="cue.label" class="flex items-center gap-1.5">
        <Icon :icon="cue.icon" class="size-4 opacity-70" />
        {{ cue.label }}
      </li>
    </ul>
    <a
      v-if="data.release"
      class="flex w-fit items-center gap-1.5 text-(--vp-c-text-3) no-underline! transition-colors hover:text-(--vp-c-brand)"
      :href="`https://github.com/amir20/dozzle/releases/tag/${data.release.version}`"
    >
      <span class="relative flex size-2">
        <span class="absolute inline-flex size-full animate-ping rounded-full bg-emerald-500 opacity-60"></span>
        <span class="relative inline-flex size-2 rounded-full bg-emerald-500"></span>
      </span>
      {{ data.release.version }} {{ released }}
    </a>
  </div>
</template>
