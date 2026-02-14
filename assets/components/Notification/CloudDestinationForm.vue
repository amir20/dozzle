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
          :class="cloudStatusError ? 'input-error' : 'input-success'"
        />
        <span class="join-item btn pointer-events-none" :class="cloudStatusError ? 'btn-error' : 'btn-success'">
          <mdi:alert-circle v-if="cloudStatusError" class="text-lg" />
          <mdi:check v-else class="text-lg" />
        </span>
      </div>

      <!-- Cloud Status -->
      <div v-if="isLoadingCloudStatus" class="mt-3 flex items-center gap-2">
        <span class="loading loading-spinner loading-sm"></span>
        <span class="text-base-content/60 text-sm">{{ $t("notifications.destination-form.cloud-checking") }}</span>
      </div>
      <div v-else-if="cloudStatusError" class="mt-3">
        <div class="alert alert-error">
          <mdi:alert-circle class="text-lg" />
          <span>{{ $t("notifications.destination-form.cloud-relink") }}</span>
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
      <button class="btn btn-primary" @click="close?.()">
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
const cloudLinkUrl = `${__CLOUD_URL__}/link?appUrl=${encodeURIComponent(callbackUrl)}`;
const cloudSettingsUrl = `${__CLOUD_URL__}/settings`;

// Cloud status
interface CloudStatus {
  user: { email: string; name: string };
  plan: { name: string; events_per_month: number; retention_days: number };
  usage: { events_used: number; events_limit: number; period: string };
}

const cloudStatus = ref<CloudStatus | null>(null);
const cloudStatusError = ref(false);
const isLoadingCloudStatus = ref(false);

const usagePercent = computed(() => {
  if (!cloudStatus.value) return 0;
  return (cloudStatus.value.usage.events_used / cloudStatus.value.usage.events_limit) * 100;
});

async function fetchCloudStatus() {
  isLoadingCloudStatus.value = true;
  cloudStatusError.value = false;
  try {
    const res = await fetch(withBase("/api/cloud/status"));
    if (!res.ok) {
      cloudStatusError.value = true;
      return;
    }
    cloudStatus.value = await res.json();
  } catch {
    cloudStatusError.value = true;
  } finally {
    isLoadingCloudStatus.value = false;
  }
}

if (destination?.prefix) {
  fetchCloudStatus();
}
</script>
