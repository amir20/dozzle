<template>
  <div class="mt-10 flex flex-wrap gap-x-14 gap-y-8">
    <a
      v-for="stat in stats"
      :key="stat.label"
      class="flex flex-col no-underline! transition-colors hover:text-(--vp-c-brand)"
      :href="stat.href"
    >
      <span class="flex items-center gap-2 text-sm text-(--vp-c-text-2)">
        <Icon :icon="stat.icon" class="size-4 opacity-70" />
        {{ stat.label }}
      </span>
      <span class="mt-2.5 text-4xl leading-none font-semibold tracking-tight text-(--vp-c-text-1) tabular-nums">
        <Counter :start="0" :end="stat.value" :duration="1000" :formatter="compact" />
      </span>
    </a>
  </div>
</template>
<script lang="ts" setup>
import { computed } from "vue";
import { data } from "../../activity.data";
import Counter from "./Counter.vue";

const stats = computed(() =>
  [
    {
      icon: "simple-icons:github",
      label: "Github Stars",
      value: data.stars,
      href: "https://github.com/amir20/dozzle/stargazers",
    },
    {
      icon: "simple-icons:docker",
      label: "Docker Pulls",
      value: data.pulls,
      href: "https://hub.docker.com/r/amir20/dozzle",
    },
    {
      icon: "mdi:account-group-outline",
      label: "Contributors",
      value: data.contributors,
      href: "https://github.com/amir20/dozzle/graphs/contributors",
    },
  ].filter((stat) => stat.value),
);

const compact = (value: number) => {
  if (value >= 1e9) return (value / 1e9).toFixed(1) + "B";
  if (value >= 1e6) return Math.round(value / 1e6) + "M";
  if (value >= 1e3) return (value / 1e3).toFixed(1) + "K";
  return Math.round(value).toString();
};
</script>
