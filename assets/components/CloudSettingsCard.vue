<template>
  <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
    <!-- Not linked -->
    <template v-if="!cloudConfig">
      <div class="flex items-start gap-4 p-4">
        <mdi:cloud class="text-base-content/40 mt-0.5 text-4xl" />
        <div class="flex flex-col gap-1">
          <p class="text-base-content/70 text-sm">{{ $t("cloud.description") }}</p>
          <div class="mt-3 flex gap-2">
            <a :href="`${cloudUrl}`" target="_blank" rel="noreferrer noopener" class="btn btn-sm">
              {{ $t("cloud.learn-more") }}
            </a>
            <a :href="cloudLinkUrl" class="btn btn-primary btn-sm">
              <mdi:link-variant class="text-base" />
              {{ $t("cloud.link-instance") }}
            </a>
          </div>
        </div>
      </div>
    </template>

    <!-- Linked -->
    <template v-else-if="cloudConfig.linked">
      <!-- Error state -->
      <div v-if="cloudStatusError" class="space-y-3 p-4">
        <div class="alert" :class="cloudStatusError === 'auth' ? 'alert-error' : 'alert-warning'">
          <mdi:alert-circle v-if="cloudStatusError === 'auth'" class="text-lg" />
          <mdi:cloud-off-outline v-else class="text-lg" />
          <span class="text-sm">{{
            cloudStatusError === "auth" ? $t("cloud.error") : $t("cloud.error-unavailable")
          }}</span>
        </div>
        <div class="flex gap-2">
          <a v-if="cloudStatusError === 'auth'" :href="cloudLinkUrl" class="btn btn-primary btn-sm">
            <mdi:link-variant class="text-base" />
            {{ $t("cloud.relink-instance") }}
          </a>
          <button v-else class="btn btn-sm" @click="fetchCloudStatus">
            <mdi:refresh class="text-base" />
            {{ $t("button.retry") }}
          </button>
          <button class="btn btn-sm btn-error" @click="confirmUnlink">
            <mdi:link-variant-off class="text-base" />
            {{ $t("cloud.unlink") }}
          </button>
        </div>
      </div>

      <!-- Loading -->
      <div v-else-if="isLoadingCloudStatus" class="flex items-center gap-2 p-4">
        <span class="loading loading-spinner loading-sm"></span>
      </div>

      <!-- Healthy -->
      <template v-else-if="cloudStatus">
        <div class="flex flex-wrap items-center gap-2 p-4">
          <span class="status-pill status-pill-success">
            <span class="size-1.5 rounded-full bg-current"></span>
            {{ $t("cloud.connected") }}
          </span>
          <span class="status-pill status-pill-primary">{{ cloudStatus.plan.name }}</span>
          <span class="text-base-content/50 text-sm">{{ cloudStatus.user.email }}</span>
        </div>

        <div class="flex flex-col gap-2 p-4">
          <div class="flex items-baseline justify-between">
            <span class="text-base-content/60 text-sm font-medium">{{ $t("cloud.usage") }}</span>
            <span class="font-mono text-sm">
              <span class="font-semibold">{{ cloudStatus.usage.events_used.toLocaleString() }}</span>
              <span class="text-base-content/40"> / {{ cloudStatus.usage.events_limit.toLocaleString() }}</span>
            </span>
          </div>
          <progress
            class="progress w-full"
            :class="usagePercent > 90 ? 'progress-error' : usagePercent > 70 ? 'progress-warning' : 'progress-primary'"
            :value="cloudStatus.usage.events_used"
            :max="cloudStatus.usage.events_limit"
          ></progress>
          <div class="text-base-content/40 flex justify-between font-mono text-xs">
            <span v-if="cloudStatus.usage.period">{{ cloudStatus.usage.period }}</span>
            <span v-else></span>
            <span>{{ usagePercent.toFixed(2) }}% used</span>
          </div>
        </div>

        <label class="flex min-h-13 cursor-pointer items-center justify-between gap-4 p-4">
          <div class="flex flex-col gap-0.5">
            <span class="text-sm font-medium">{{ $t("cloud.stream-logs") }}</span>
            <span class="text-base-content/60 text-xs">{{ $t("cloud.stream-logs-help") }}</span>
          </div>
          <input
            type="checkbox"
            class="toggle toggle-primary toggle-sm shrink-0"
            :checked="streamLogs"
            :disabled="isSavingStreamLogs"
            @change="onStreamLogsChange(($event.target as HTMLInputElement).checked)"
          />
        </label>

        <div class="flex gap-2 p-4">
          <a :href="cloudUrl" target="_blank" rel="noreferrer noopener" class="btn btn-sm">
            {{ $t("cloud.dashboard") }}
          </a>
          <button class="btn btn-sm btn-error" @click="confirmUnlink">
            {{ $t("cloud.unlink") }}
          </button>
        </div>
      </template>
    </template>

    <!-- Unlink confirmation modal -->
    <dialog ref="unlinkModal" class="modal">
      <div class="modal-box">
        <h3 class="text-lg font-bold">{{ $t("cloud.unlink") }}</h3>
        <p class="py-4 text-sm">{{ $t("cloud.unlink-confirm") }}</p>
        <div class="modal-action">
          <form method="dialog">
            <button class="btn btn-sm">{{ $t("button.cancel") }}</button>
          </form>
          <button class="btn btn-error btn-sm" :disabled="isUnlinking" @click="doUnlink">
            <span v-if="isUnlinking" class="loading loading-spinner loading-xs"></span>
            {{ $t("cloud.unlink") }}
          </button>
        </div>
      </div>
      <form method="dialog" class="modal-backdrop">
        <button></button>
      </form>
    </dialog>
  </div>
</template>

<script lang="ts" setup>
const cloudUrl = __CLOUD_URL__;
const callbackUrl = `${window.location.origin}${withBase("/")}`;
const cloudLinkUrl = `${cloudUrl}/link?appUrl=${encodeURIComponent(callbackUrl)}&from=cloud`;

const {
  cloudConfig,
  cloudStatus,
  cloudStatusError,
  isLoadingCloudStatus,
  initialLoad,
  fetchCloudStatus,
  clearCloudState,
} = useCloudConfig();
const isUnlinking = ref(false);
const unlinkModal = ref<HTMLDialogElement | null>(null);

const streamLogs = ref(true);
const isSavingStreamLogs = ref(false);
watchEffect(() => {
  if (cloudConfig.value) streamLogs.value = cloudConfig.value.streamLogs;
});

async function onStreamLogsChange(value: boolean | undefined) {
  if (!cloudConfig.value || value === undefined) return;
  isSavingStreamLogs.value = true;
  try {
    const res = await fetch(withBase("/api/cloud/config"), {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ streamLogs: value }),
    });
    if (!res.ok) {
      streamLogs.value = !value;
      return;
    }
    cloudConfig.value.streamLogs = value;
  } catch {
    streamLogs.value = !value;
  } finally {
    isSavingStreamLogs.value = false;
  }
}

const usagePercent = computed(() => {
  if (!cloudStatus.value) return 0;
  return (cloudStatus.value.usage.events_used / cloudStatus.value.usage.events_limit) * 100;
});

function confirmUnlink() {
  unlinkModal.value?.showModal();
}

async function doUnlink() {
  isUnlinking.value = true;
  try {
    const res = await fetch(withBase("/api/cloud/config"), { method: "DELETE" });
    if (!res.ok) {
      cloudStatusError.value = "unavailable";
      return;
    }
    clearCloudState();
    unlinkModal.value?.close();
  } finally {
    isUnlinking.value = false;
  }
}

onMounted(async () => {
  await initialLoad;
  if (cloudConfig.value?.linked) {
    fetchCloudStatus();
  }
});
</script>
