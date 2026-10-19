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

    <InlineNotice v-if="error" type="error" class="mt-4">{{ error }}</InlineNotice>

    <div class="mt-4 flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
      <p class="text-base-content/40 text-xs">{{ $t("setup.cloud.first-week") }}</p>
      <a
        :href="cloudUrl"
        target="_blank"
        rel="noreferrer noopener"
        class="link link-hover text-base-content/60 inline-block text-sm"
      >
        {{ $t("setup.cloud.learn-more") }}
        <mdi:open-in-new class="inline size-3.5 align-[-0.1em] opacity-40" />
      </a>
    </div>
  </div>
</template>

<script lang="ts" setup>
import MdiBellRingOutline from "~icons/mdi/bell-ring-outline";
import MdiChatProcessingOutline from "~icons/mdi/chat-processing-outline";
import MdiRobotOutline from "~icons/mdi/robot-outline";
import type { SetupNextResult, SetupStepId } from "@/composable/setup/setup";

const { nextStep } = defineProps<{ nextStep?: SetupStepId }>();

const { t } = useI18n();

const cloudUrl = config.cloudUrl;

// The same three things the cloud popover leads with, in the same order.
const values = computed(() => [
  { key: "report", icon: MdiRobotOutline, title: t("setup.cloud.report-title"), body: t("setup.cloud.report-body") },
  { key: "alerts", icon: MdiBellRingOutline, title: t("setup.cloud.alerts-title"), body: t("setup.cloud.alerts-body") },
  { key: "chat", icon: MdiChatProcessingOutline, title: t("setup.cloud.chat-title"), body: t("setup.cloud.chat-body") },
]);

const busy = ref(false);
const error = ref("");

// The link leaves the app. The marker makes the wizard, not the cloud welcome
// modal, own the #cloudLinked return.
async function next(): Promise<SetupNextResult> {
  error.value = "";
  busy.value = true;
  writeSetupResume(nextStep ?? "restart");
  try {
    // Same link flow as CloudPopover, tagged so Cloud can tell where the link came from.
    await startCloudLink("setup");
  } catch {
    // Nothing left the page, so there is nothing to resume.
    clearSetupResume();
    error.value = t("setup.error.generic");
    busy.value = false;
  }
  // On success the page is already navigating away; busy stays on until it does.
  return "stay";
}

const nextLabel = computed(() => t("setup.cloud.connect"));
const skipLabel = computed(() => t("setup.cloud.skip"));
const nextDisabled = computed(() => busy.value);

// Connect is the highlighted answer, as Login's is. "Not now" sits beside it at the
// same size, so declining stays as easy to find.
defineExpose({ nextLabel, nextDisabled, nextPlain: false, skipLabel, busy, next });
</script>
