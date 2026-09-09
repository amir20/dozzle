<template>
  <div class="space-y-4">
    <!-- Cloud linked (when editing with prefix) -->
    <fieldset v-if="destination?.prefix" class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.destination-form.api-key") }}</legend>
      <div class="join w-full">
        <input
          type="text"
          :value="destination.prefix + '**************************************'"
          readonly
          disabled
          class="input join-item w-full font-mono"
          :class="
            cloudStatusError === 'auth'
              ? 'input-error'
              : cloudStatusError === 'unavailable'
                ? 'input-warning'
                : 'input-success'
          "
        />
        <!-- Tinted glyph rather than a filled button: the state is already on the
             input's border, and a solid red block on the end of a masked key read
             as something you were meant to press. -->
        <span
          class="join-item btn pointer-events-none"
          :class="
            cloudStatusError === 'auth'
              ? 'text-error'
              : cloudStatusError === 'unavailable'
                ? 'text-warning'
                : 'text-success'
          "
        >
          <mdi:alert-circle v-if="cloudStatusError === 'auth'" class="text-lg" />
          <mdi:cloud-off-outline v-else-if="cloudStatusError === 'unavailable'" class="text-lg" />
          <mdi:check v-else class="text-lg" />
        </span>
      </div>

      <!-- Cloud Status -->
      <div v-if="isLoadingCloudStatus" class="mt-3 flex items-center gap-2">
        <span class="loading loading-spinner loading-sm"></span>
        <span class="text-base-content/60 text-sm">{{ $t("notifications.destination-form.cloud-checking") }}</span>
      </div>
      <!-- Severity rides on the icon, not on a full-width saturated bar: the drawer
           is the width of the page and that block shouted over the key above it. -->
      <div v-else-if="cloudStatusError" class="mt-3 flex items-start gap-3">
        <div
          class="shrink-0 rounded-full p-1.5"
          :class="cloudStatusError === 'auth' ? 'bg-error/10 text-error' : 'bg-warning/10 text-warning'"
        >
          <mdi:alert-circle-outline v-if="cloudStatusError === 'auth'" class="size-5" />
          <mdi:cloud-off-outline v-else class="size-5" />
        </div>
        <div class="flex min-w-0 flex-col items-start gap-3">
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
      </div>
      <div v-else-if="cloudStatus" class="mt-3 space-y-3">
        <div class="flex items-center justify-between text-sm">
          <span class="text-base-content/60">{{ $t("notifications.destination-form.cloud-plan") }}</span>
          <span class="badge badge-primary badge-sm capitalize">{{ cloudStatus.plan.name }}</span>
        </div>
        <div>
          <div class="mb-1 flex items-center justify-between text-sm">
            <span class="text-base-content/60">{{ $t("notifications.destination-form.cloud-usage") }}</span>
            <span
              >{{ cloudStatus.usage.events_used.toLocaleString() }} /
              {{ cloudStatus.usage.events_limit.toLocaleString() }}</span
            >
          </div>
          <progress
            class="progress w-full"
            :class="usagePercent > 90 ? 'progress-error' : usagePercent > 70 ? 'progress-warning' : 'progress-primary'"
            :value="cloudStatus.usage.events_used"
            :max="cloudStatus.usage.events_limit"
          ></progress>
        </div>
      </div>

      <p class="text-base-content/60 mt-2 text-sm">
        {{ $t("notifications.destination-form.cloud-settings-hint") }}
        <a :href="cloudSettingsUrl" target="_blank" class="link link-primary">
          {{ $t("notifications.destination-form.cloud-settings-link") }}
        </a>
      </p>
    </fieldset>

    <!-- Link Dozzle Cloud (when creating or not linked) -->
    <div v-else class="card card-border border-primary/30 bg-primary/5">
      <div class="card-body items-center text-center">
        <mdi:cloud-outline class="text-primary text-4xl" />
        <h3 class="card-title">{{ $t("notifications.destination-form.link-cloud") }}</h3>
        <p class="text-base-content/60 text-sm">{{ $t("notifications.destination-form.cloud-description") }}</p>
        <a :href="cloudLinkUrl" class="btn btn-primary btn-lg mt-2">
          <mdi:link-variant class="text-lg" />
          {{ $t("notifications.destination-form.link-cloud-button") }}
        </a>
      </div>
    </div>

    <!-- Actions -->
    <div class="flex items-center gap-2 pt-4">
      <div class="flex-1"></div>
      <button class="btn" @click="close?.()">
        {{ $t("notifications.destination-form.close") }}
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { Dispatcher } from "@/types/notifications";

const { destination, close } = defineProps<{
  destination?: Dispatcher;
  close?: () => void;
}>();

const callbackUrl = `${window.location.origin}${withBase("/")}`;
const cloudLinkUrl = `${config.cloudUrl}/link?appUrl=${encodeURIComponent(callbackUrl)}&from=notifications`;
const cloudSettingsUrl = `${config.cloudUrl}/settings`;

const { cloudStatus, cloudStatusError, isLoadingCloudStatus, fetchCloudStatus } = useCloudConfig();

const usagePercent = computed(() => {
  if (!cloudStatus.value) return 0;
  return (cloudStatus.value.usage.events_used / cloudStatus.value.usage.events_limit) * 100;
});

if (destination?.prefix) {
  fetchCloudStatus();
}
</script>
