<template>
  <div class="mt-10 flex justify-center gap-x-6 min-[960px]:justify-start min-[960px]:gap-x-14">
    <a
      v-for="stat in stats"
      :key="stat.label"
      class="flex flex-col items-center no-underline! transition-colors hover:text-(--vp-c-brand) min-[960px]:items-start"
      :href="stat.href"
    >
      <span class="flex items-center gap-1.5 text-xs whitespace-nowrap text-(--vp-c-text-2) sm:gap-2 sm:text-sm">
        <Icon :icon="stat.icon" class="size-4 shrink-0 opacity-70" />
        {{ stat.label }}
      </span>
      <span
        class="mt-2.5 text-3xl leading-none font-semibold tracking-tight text-(--vp-c-text-1) tabular-nums sm:text-4xl"
      >
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
