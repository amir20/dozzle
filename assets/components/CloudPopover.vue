<template>
  <Dropdown class="dropdown-end" @click="onOpen">
    <template #trigger>
      <div class="relative">
        <mdi:cloud
          class="size-6"
          :class="
            !cloudConfig
              ? 'text-base-content/40'
              : cloudConfig.linked && !cloudStatusError
                ? 'text-info'
                : cloudStatusError === 'unavailable'
                  ? 'text-warning'
                  : 'text-error'
          "
        />
        <span
          v-if="cloudConfig?.linked"
          class="absolute -top-0.5 -right-0.5 size-2 rounded-full"
          :class="
            cloudStatusError === 'auth'
              ? 'bg-error'
              : cloudStatusError === 'unavailable'
                ? 'bg-warning'
                : cloudStatusError
                  ? 'bg-error'
                  : 'bg-success'
          "
        ></span>
      </div>
    </template>
    <template #content>
      <div class="w-80 space-y-3 p-1">
        <!-- Not linked -->
        <template v-if="!cloudConfig">
          <div class="flex flex-col items-center gap-2 p-2 text-center">
            <mdi:cloud class="text-base-content/40 text-4xl" />
            <h3 class="text-base font-bold">{{ $t("cloud.title") }}</h3>
            <p class="text-base-content/60 text-sm">{{ $t("cloud.description") }}</p>
            <div class="mt-2 flex w-full gap-2">
              <a :href="`${cloudUrl}`" target="_blank" rel="noreferrer noopener" class="btn btn-sm flex-1">
                {{ $t("cloud.learn-more") }}
              </a>
              <a :href="cloudLinkUrl" class="btn btn-primary btn-sm flex-1">
                <mdi:link-variant class="text-base" />
                {{ $t("cloud.link-instance") }}
              </a>
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
            <a v-if="cloudStatusError === 'auth'" :href="cloudLinkUrl" class="btn btn-primary btn-sm w-full">
              <mdi:link-variant class="text-base" />
              {{ $t("cloud.relink-instance") }}
            </a>
            <button v-else class="btn btn-sm w-full" @click="fetchCloudStatus">
              <mdi:refresh class="text-base" />
              {{ $t("button.retry") }}
            </button>
          </div>

          <!-- Loading -->
          <div v-else-if="isLoadingCloudStatus" class="flex items-center justify-center gap-2 py-4">
            <span class="loading loading-spinner loading-xs"></span>
          </div>

          <!-- Healthy -->
          <div v-else-if="cloudStatus" class="space-y-3">
            <div class="flex items-center justify-between">
              <h3 class="font-bold">{{ $t("cloud.title") }}</h3>
              <div class="flex items-center gap-1">
                <span class="badge badge-success badge-sm">{{ $t("cloud.connected") }}</span>
                <span class="badge badge-primary badge-sm capitalize">{{ cloudStatus.plan.name }}</span>
              </div>
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
                class="progress w-full"
                :class="
                  usagePercent > 90 ? 'progress-error' : usagePercent > 70 ? 'progress-warning' : 'progress-primary'
                "
                :value="cloudStatus.usage.events_used"
                :max="cloudStatus.usage.events_limit"
              ></progress>
            </div>

            <div class="flex gap-2">
              <a :href="cloudUrl" target="_blank" rel="noreferrer noopener" class="btn btn-sm flex-1">
                {{ $t("cloud.dashboard") }}
              </a>
              <a :href="`${cloudUrl}/settings`" target="_blank" rel="noreferrer noopener" class="btn btn-sm flex-1">
                {{ $t("cloud.settings") }}
              </a>
            </div>
          </div>
        </template>
      </div>
    </template>
  </Dropdown>
  <WelcomeModal ref="welcomeModal" />
</template>

<script lang="ts" setup>
const cloudUrl = __CLOUD_URL__;
const callbackUrl = `${window.location.origin}${withBase("/")}`;
const cloudLinkUrl = `${cloudUrl}/link?appUrl=${encodeURIComponent(callbackUrl)}&from=cloud`;

const { cloudConfig, cloudStatus, cloudStatusError, isLoadingCloudStatus, fetchCloudConfig, fetchCloudStatus } =
  useCloudConfig();

const welcomeModal = ref<{ open: () => void }>();
const cloudWelcomeShown = useProfileStorage("cloudWelcomeShown", false);

const usagePercent = computed(() => {
  if (!cloudStatus.value) return 0;
  return (cloudStatus.value.usage.events_used / cloudStatus.value.usage.events_limit) * 100;
});

function onOpen() {
  if (cloudConfig.value?.linked && !cloudStatus.value && !isLoadingCloudStatus.value) {
    fetchCloudStatus();
  }
}

onMounted(async () => {
  await fetchCloudConfig();

  if (cloudConfig.value?.linked) {
    fetchCloudStatus();
  }

  // Handle successful OAuth return — show welcome modal
  if (window.location.hash === "#cloudLinked" && !cloudWelcomeShown.value) {
    cloudWelcomeShown.value = true;
    nextTick(() => welcomeModal.value?.open());
    history.replaceState(history.state, "", window.location.pathname + window.location.search);
  }
});
</script>
