<template>
  <div>
    <!-- Not linked -->
    <template v-if="!cloudConfig">
      <div class="flex items-start gap-4">
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
      <div v-if="cloudStatusError" class="space-y-3">
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
      <div v-else-if="isLoadingCloudStatus" class="flex items-center gap-2 py-2">
        <span class="loading loading-spinner loading-sm"></span>
      </div>

      <!-- Healthy -->
      <div v-else-if="cloudStatus" class="space-y-4">
        <div class="flex items-center gap-2">
          <span class="badge badge-success">{{ $t("cloud.connected") }}</span>
          <span class="badge badge-primary capitalize">{{ cloudStatus.plan.name }}</span>
          <span class="text-base-content/50 text-sm">{{ cloudStatus.user.email }}</span>
        </div>

        <div>
          <div class="mb-1 flex items-center justify-between text-sm">
            <span class="text-base-content/60">{{ $t("cloud.usage") }}</span>
            <span>
              {{ cloudStatus.usage.events_used.toLocaleString() }} /
              {{ cloudStatus.usage.events_limit.toLocaleString() }}
            </span>
          </div>
          <progress
            class="progress w-full max-w-xs"
            :class="usagePercent > 90 ? 'progress-error' : usagePercent > 70 ? 'progress-warning' : 'progress-primary'"
            :value="cloudStatus.usage.events_used"
            :max="cloudStatus.usage.events_limit"
          ></progress>
        </div>

        <div class="flex gap-2">
          <a :href="cloudUrl" target="_blank" rel="noreferrer noopener" class="btn btn-sm">
            {{ $t("cloud.dashboard") }}
          </a>
          <button class="btn btn-sm btn-error" @click="confirmUnlink">
            {{ $t("cloud.unlink") }}
          </button>
        </div>
      </div>
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
  fetchCloudConfig,
  fetchCloudStatus,
  clearCloudState,
} = useCloudConfig();
const isUnlinking = ref(false);
const unlinkModal = ref<HTMLDialogElement | null>(null);

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
  await fetchCloudConfig();
  if (cloudConfig.value?.linked) {
    fetchCloudStatus();
  }
});
</script>
