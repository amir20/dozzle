<template>
  <div class="flex min-h-full flex-1 flex-col">
    <!--
      Linked. Nothing here is editable, so the drawer stops pretending to be a form:
      no fieldset legend, no disabled input holding a key you cannot change. Same parts
      as the cloud popover and the settings card (identity row, hairline dividers,
      meter, link rows) so the three read as one surface at three widths.

      Full bleed rather than a bordered card: a rounded panel inset inside a drawer that
      is already a panel draws a box around the drawer's only content and leaves a margin
      of nothing around it. The rows run to the edges the way the footer below does, and
      the dividers alone carry the grouping.
    -->
    <div v-if="destination?.prefix" class="divide-base-content/10 border-base-content/10 -mx-4 divide-y border-y">
      <!--
        Identity leads with the account rather than the product name: the drawer
        header two lines up already says Dozzle Cloud, and what you cannot tell
        from there is which account this key belongs to.
      -->
      <div class="flex items-center gap-3 p-4">
        <div class="shrink-0 rounded-full p-2" :class="accent.tint">
          <mdi:alert-circle-outline v-if="cloudStatusError === 'auth'" class="size-6" />
          <mdi:cloud-off-outline v-else-if="cloudStatusError === 'unavailable'" class="size-6" />
          <mdi:cloud v-else class="size-6" />
        </div>
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-1.5">
            <span class="truncate font-semibold">{{ cloudStatus?.user.email ?? $t("cloud.title") }}</span>
            <span class="size-1.5 shrink-0 rounded-full" :class="accent.dot" :title="statusLabel"></span>
          </div>
          <!-- An error says its piece in the body below at full length; repeating a
               truncated copy of it here would only lose half the sentence. -->
          <div v-if="!cloudStatusError" class="text-base-content/60 truncate text-sm">{{ statusLabel }}</div>
        </div>
        <span v-if="cloudStatus" class="status-pill status-pill-primary shrink-0 capitalize">
          {{ cloudStatus.plan.name }}
        </span>
      </div>

      <!-- The key is a fact about the link, not an input: label left, masked value right. -->
      <div class="flex items-center justify-between gap-4 p-4">
        <span class="text-base-content/60 text-sm">{{ $t("notifications.destination-form.api-key") }}</span>
        <span class="truncate font-mono text-sm">
          {{ destination.prefix }}<span class="text-base-content/30 tracking-widest">••••••••••••</span>
        </span>
      </div>

      <!-- The identity row already says the check is running, so this is just a placeholder
           holding the meter's height so the panel does not jump when it arrives. -->
      <div v-if="isLoadingCloudStatus" class="flex items-center justify-center p-4">
        <span class="loading loading-spinner loading-sm"></span>
      </div>

      <!--
        Severity rides on the icon above and the message here; the surface stays
        neutral, which keeps the text at full contrast instead of washed onto a
        saturated bar the width of the drawer.
      -->
      <div v-else-if="cloudStatusError" class="flex flex-col items-start gap-3 p-4">
        <p class="text-sm">
          {{
            cloudStatusError === "auth"
              ? $t("notifications.destination-form.cloud-relink")
              : $t("notifications.destination-form.cloud-unavailable")
          }}
        </p>
        <a v-if="cloudStatusError === 'auth'" :href="cloudLinkUrl" class="btn btn-primary btn-sm">
          <mdi:link-variant class="text-base" />
          {{ $t("cloud.relink-instance") }}
        </a>
        <button v-else class="btn btn-sm" @click="fetchCloudStatus">
          <mdi:refresh class="text-base" />
          {{ $t("button.retry") }}
        </button>
      </div>

      <div v-else-if="cloudStatus" class="p-4">
        <CloudUsage :usage="cloudStatus.usage" />
      </div>

      <!-- Managed channels live on the cloud side, so the drawer ends in a way out to them. -->
      <div class="p-2">
        <a
          :href="cloudSettingsUrl"
          target="_blank"
          rel="noreferrer noopener"
          class="hover:bg-base-300 flex items-center gap-2 rounded-md px-2 py-2 text-sm transition-colors"
        >
          <mdi:cog-outline class="size-4 opacity-60" />
          <span class="flex-1">{{ $t("notifications.destination-form.cloud-settings-link") }}</span>
          <mdi:open-in-new class="size-3.5 opacity-40" />
        </a>
        <a
          :href="cloudUrl"
          target="_blank"
          rel="noreferrer noopener"
          class="hover:bg-base-300 flex items-center gap-2 rounded-md px-2 py-2 text-sm transition-colors"
        >
          <mdi:view-dashboard-outline class="size-4 opacity-60" />
          <span class="flex-1">{{ $t("cloud.dashboard") }}</span>
          <mdi:open-in-new class="size-3.5 opacity-40" />
        </a>
      </div>
    </div>

    <!--
      Not linked. Same shell as the linked state, so linking does not reshuffle the
      drawer: the pitch rows are replaced by the key and the meter. The header above
      already carries the title and the pitch sentence, so this branch opens on the
      three concrete things you get rather than restating them.
    -->
    <div v-else class="divide-base-content/10 border-base-content/10 -mx-4 divide-y border-y">
      <ul class="text-base-content/70 space-y-2.5 p-4 text-sm">
        <li class="flex items-start gap-2">
          <mdi:robot-outline class="text-info mt-0.5 size-4 shrink-0" />
          <span>{{ $t("cloud.pitch.findings") }}</span>
        </li>
        <li class="flex items-start gap-2">
          <mdi:bell-ring-outline class="text-info mt-0.5 size-4 shrink-0" />
          <span>{{ $t("cloud.pitch.alerts") }}</span>
        </li>
        <li class="flex items-start gap-2">
          <mdi:remote class="text-info mt-0.5 size-4 shrink-0" />
          <span>{{ $t("cloud.pitch.control") }}</span>
        </li>
      </ul>

      <div class="flex gap-2 p-4">
        <a :href="cloudUrl" target="_blank" rel="noreferrer noopener" class="btn btn-sm">
          {{ $t("cloud.learn-more") }}
        </a>
        <a :href="cloudLinkUrl" class="btn btn-primary btn-sm">
          <mdi:link-variant class="text-base" />
          {{ $t("notifications.destination-form.link-cloud-button") }}
        </a>
      </div>
    </div>

    <!-- Actions. Same sticky, full-bleed bar as the webhook form next door. -->
    <div class="bg-base-100 border-base-content/10 sticky bottom-0 z-10 -mx-4 mt-auto border-t px-4 py-4">
      <div class="flex items-center justify-end">
        <button class="btn" @click="close?.()">
          {{ $t("notifications.destination-form.close") }}
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { Dispatcher } from "@/types/notifications";

const { destination, close } = defineProps<{
  destination?: Dispatcher;
  close?: () => void;
}>();

const { t } = useI18n();

const cloudUrl = config.cloudUrl;
const callbackUrl = `${window.location.origin}${withBase("/")}`;
const cloudLinkUrl = `${cloudUrl}/link?appUrl=${encodeURIComponent(callbackUrl)}&from=notifications`;
const cloudSettingsUrl = `${cloudUrl}/settings`;

const { cloudStatus, cloudStatusError, isLoadingCloudStatus, fetchCloudStatus } = useCloudConfig();

const accent = computed(() => {
  if (cloudStatusError.value === "auth") return { tint: "bg-error/10 text-error", dot: "bg-error" };
  if (cloudStatusError.value === "unavailable") return { tint: "bg-warning/10 text-warning", dot: "bg-warning" };
  if (!cloudStatus.value) return { tint: "bg-info/10 text-info", dot: "bg-base-content/30" };
  return { tint: "bg-info/10 text-info", dot: "bg-success" };
});

const statusLabel = computed(() => {
  if (cloudStatusError.value === "auth") return t("cloud.error");
  if (cloudStatusError.value === "unavailable") return t("cloud.error-unavailable");
  // Before the first check comes back the dot is grey and this says so, rather than
  // claiming a connection that has not been confirmed yet.
  if (!cloudStatus.value) return t("notifications.destination-form.cloud-checking");
  return t("cloud.connected");
});

if (destination?.prefix) {
  fetchCloudStatus();
}
</script>
