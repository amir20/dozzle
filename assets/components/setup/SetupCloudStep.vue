<template>
  <div>
    <h2 class="text-2xl font-bold">{{ $t("setup.cloud.title") }}</h2>
    <p class="text-base-content/60 mt-1 text-sm">{{ $t("setup.cloud.subtitle") }}</p>

    <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 mt-6 divide-y rounded-lg border">
      <div v-for="item in values" :key="item.key" class="flex items-start gap-3 p-4">
        <div class="bg-info/10 text-info shrink-0 rounded-full p-2">
          <component :is="item.icon" class="size-5" />
        </div>
        <div class="min-w-0">
          <div class="text-sm font-semibold">{{ item.title }}</div>
          <p class="text-base-content/60 mt-0.5 text-sm">{{ item.body }}</p>
        </div>
      </div>
    </div>

    <a
      :href="cloudUrl"
      target="_blank"
      rel="noreferrer noopener"
      class="link link-hover text-base-content/60 mt-4 inline-block text-sm"
    >
      {{ $t("setup.cloud.learn-more") }}
      <mdi:open-in-new class="inline size-3.5 align-[-0.1em] opacity-40" />
    </a>
  </div>
</template>

<script lang="ts" setup>
import MdiBellRingOutline from "~icons/mdi/bell-ring-outline";
import MdiWeatherSunsetUp from "~icons/mdi/weather-sunset-up";
import MdiHistory from "~icons/mdi/history";
import type { SetupNextResult, SetupStepId } from "@/composable/setup/setup";

const { nextStep } = defineProps<{ nextStep?: SetupStepId }>();

const { t } = useI18n();

const cloudUrl = config.cloudUrl;
const callbackUrl = `${window.location.origin}${withBase("/")}`;
// Same link flow as CloudPopover, tagged so Cloud can tell where the link came from.
const linkUrl = `${cloudUrl}/link?appUrl=${encodeURIComponent(callbackUrl)}&from=setup`;

const values = computed(() => [
  { key: "alerts", icon: MdiBellRingOutline, title: t("setup.cloud.alerts-title"), body: t("setup.cloud.alerts-body") },
  {
    key: "summary",
    icon: MdiWeatherSunsetUp,
    title: t("setup.cloud.summary-title"),
    body: t("setup.cloud.summary-body"),
  },
  { key: "history", icon: MdiHistory, title: t("setup.cloud.history-title"), body: t("setup.cloud.history-body") },
]);

// The link leaves the app. The marker makes the wizard, not the cloud welcome
// modal, own the #cloudLinked return.
async function next(): Promise<SetupNextResult> {
  writeSetupResume(nextStep ?? "restart");
  window.location.assign(linkUrl);
  return "stay";
}

const nextLabel = computed(() => t("setup.cloud.connect"));
const skipLabel = computed(() => t("setup.cloud.skip"));

// Connect stays a plain button: Cloud is optional, so neither answer is the highlighted one.
defineExpose({ nextLabel, nextDisabled: false, nextPlain: true, skipLabel, busy: false, next });
</script>
