<template>
  <Dropdown class="dropdown-end" @click="onOpen">
    <template #trigger>
      <div class="relative">
        <mdi:cloud
          class="icon-float size-6"
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
      <div class="w-76">
        <!--
          Not linked. Same header shape as the linked state so the panel doesn't
          reshuffle after linking. Three concrete rows sell it better than the
          one abstract sentence that used to sit here, so `cloud.description` is
          left to the settings card where there is room for prose.
        -->
        <template v-if="!cloudConfig">
          <div class="flex items-center gap-2.5 px-2 py-1.5">
            <div class="bg-info/10 text-info shrink-0 rounded-full p-1.5">
              <mdi:cloud class="size-5" />
            </div>
            <span class="text-sm font-semibold">{{ $t("cloud.title") }}</span>
          </div>

          <div class="bg-base-content/10 my-1.5 h-px"></div>

          <ul class="text-base-content/70 space-y-2 px-2 py-1 text-xs">
            <li class="flex items-start gap-2">
              <mdi:robot-outline class="text-info mt-px size-4 shrink-0" />
              <span>{{ $t("cloud.pitch.findings") }}</span>
            </li>
            <li class="flex items-start gap-2">
              <mdi:bell-ring-outline class="text-info mt-px size-4 shrink-0" />
              <span>{{ $t("cloud.pitch.alerts") }}</span>
            </li>
            <li class="flex items-start gap-2">
              <mdi:remote class="text-info mt-px size-4 shrink-0" />
              <span>{{ $t("cloud.pitch.control") }}</span>
            </li>
          </ul>

          <div class="bg-base-content/10 my-1.5 h-px"></div>

          <div class="flex gap-2 px-1 pb-0.5">
            <a :href="`${cloudUrl}`" target="_blank" rel="noreferrer noopener" class="btn btn-sm flex-1">
              {{ $t("cloud.learn-more") }}
            </a>
            <a :href="cloudLinkUrl" class="btn btn-primary btn-sm flex-1">
              <mdi:link-variant class="text-base" />
              {{ $t("cloud.link-instance") }}
            </a>
          </div>
        </template>

        <!-- Linked -->
        <template v-else-if="cloudConfig.linked">
          <!-- Error state -->
          <div v-if="cloudStatusError" class="space-y-3 p-1">
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
          <div v-else-if="isLoadingCloudStatus" class="flex items-center justify-center gap-2 py-6">
            <span class="loading loading-spinner loading-xs"></span>
          </div>

          <!-- Healthy -->
          <div v-else-if="cloudStatus">
            <!--
              Header mirrors the user menu next door: avatar-ish icon, one min-w-0
              identity column so a long email truncates instead of widening the
              panel, and the badge pinned right.
            -->
            <div class="flex items-center gap-2.5 px-2 py-1.5">
              <div class="bg-info/10 text-info shrink-0 rounded-full p-1.5">
                <mdi:cloud class="size-5" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-1.5">
                  <span class="truncate text-sm font-semibold">{{ $t("cloud.title") }}</span>
                  <span class="bg-success size-1.5 shrink-0 rounded-full" :title="$t('cloud.connected')"></span>
                </div>
                <div class="text-base-content/60 truncate text-xs">{{ cloudStatus.user.email }}</div>
              </div>
              <span class="status-pill status-pill-primary shrink-0 capitalize">{{ cloudStatus.plan.name }}</span>
            </div>

            <div class="bg-base-content/10 my-1.5 h-px"></div>

            <div class="flex flex-col gap-1.5 px-2 py-1">
              <div class="flex items-baseline justify-between gap-2">
                <span class="text-base-content/60 text-xs">{{ $t("cloud.usage") }}</span>
                <span class="font-mono text-xs">
                  <span class="font-semibold">{{ cloudStatus.usage.events_used.toLocaleString() }}</span>
                  <span class="text-base-content/40"> / {{ cloudStatus.usage.events_limit.toLocaleString() }}</span>
                </span>
              </div>
              <!--
                A daisyUI <progress> is taller than the type it sits under here, so
                the meter is a plain div pair scaled to the row instead.
              -->
              <div class="bg-base-content/10 h-1.5 w-full overflow-hidden rounded-full">
                <div
                  class="h-full rounded-full transition-[width] duration-500"
                  :class="usagePercent > 90 ? 'bg-error' : usagePercent > 70 ? 'bg-warning' : 'bg-primary'"
                  :style="{ width: `${Math.min(usagePercent, 100)}%` }"
                ></div>
              </div>
              <div class="text-base-content/40 flex justify-between gap-2 font-mono text-[0.6875rem]">
                <span class="truncate">{{ cloudStatus.usage.period }}</span>
                <span class="shrink-0">{{ usagePercent.toFixed(1) }}%</span>
              </div>
            </div>

            <div class="bg-base-content/10 my-1.5 h-px"></div>

            <a
              :href="cloudUrl"
              target="_blank"
              rel="noreferrer noopener"
              class="hover:bg-base-300 flex items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors"
            >
              <mdi:view-dashboard-outline class="size-4 opacity-60" />
              <span class="flex-1">{{ $t("cloud.dashboard") }}</span>
              <mdi:open-in-new class="size-3.5 opacity-40" />
            </a>
            <a
              :href="`${cloudUrl}/settings`"
              target="_blank"
              rel="noreferrer noopener"
              class="hover:bg-base-300 flex items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors"
            >
              <mdi:cog-outline class="size-4 opacity-60" />
              <span class="flex-1">{{ $t("cloud.settings") }}</span>
              <mdi:open-in-new class="size-3.5 opacity-40" />
            </a>
          </div>
        </template>
      </div>
    </template>
  </Dropdown>
  <WelcomeModal ref="welcomeModal" />
</template>

<script lang="ts" setup>
const cloudUrl = config.cloudUrl;
const callbackUrl = `${window.location.origin}${withBase("/")}`;
const cloudLinkUrl = `${cloudUrl}/link?appUrl=${encodeURIComponent(callbackUrl)}&from=cloud`;

const {
  cloudConfig,
  cloudStatus,
  cloudStatusError,
  isLoadingCloudStatus,
  initialLoad,
  fetchCloudStatus,
  ensureCloudStatus,
} = useCloudConfig();

const welcomeModal = ref<{ open: () => void }>();
const cloudWelcomeShown = useProfileStorage("cloudWelcomeShown", false);

const usagePercent = computed(() => {
  if (!cloudStatus.value) return 0;
  const { events_used, events_limit } = cloudStatus.value.usage;
  if (!events_limit) return 0;
  return (events_used / events_limit) * 100;
});

function onOpen() {
  ensureCloudStatus();
}

onMounted(async () => {
  await initialLoad;
  ensureCloudStatus();

  // Handle successful OAuth return — show welcome modal
  if (window.location.hash === "#cloudLinked" && !cloudWelcomeShown.value) {
    cloudWelcomeShown.value = true;
    nextTick(() => welcomeModal.value?.open());
    history.replaceState(history.state, "", window.location.pathname + window.location.search);
  }
});
</script>
