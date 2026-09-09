<template>
  <Dropdown placement="bottom-end" @opened="onOpen">
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
          <!--
            Error state. Same header/divider/footer shape as the other two
            branches so the panel keeps its size when the connection drops.
            Severity rides on the icon and the status dot; the surface stays
            neutral, which leaves the message at full contrast instead of
            washed onto a saturated block.
          -->
          <div v-if="cloudStatusError">
            <div class="flex items-start gap-2.5 px-2 py-1.5">
              <div
                class="shrink-0 rounded-full p-1.5"
                :class="cloudStatusError === 'auth' ? 'bg-error/10 text-error' : 'bg-warning/10 text-warning'"
              >
                <mdi:alert-circle-outline v-if="cloudStatusError === 'auth'" class="size-5" />
                <mdi:cloud-off-outline v-else class="size-5" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-1.5">
                  <span class="truncate text-sm font-semibold">{{ $t("cloud.title") }}</span>
                  <span
                    class="size-1.5 shrink-0 rounded-full"
                    :class="cloudStatusError === 'auth' ? 'bg-error' : 'bg-warning'"
                  ></span>
                </div>
                <div class="text-base-content/60 text-xs leading-relaxed">
                  {{ cloudStatusError === "auth" ? $t("cloud.error") : $t("cloud.error-unavailable") }}
                </div>
              </div>
            </div>

            <div class="bg-base-content/10 my-1.5 h-px"></div>

            <div class="flex gap-2 px-1 pb-0.5">
              <a v-if="cloudStatusError === 'auth'" :href="cloudLinkUrl" class="btn btn-primary btn-sm flex-1">
                <mdi:link-variant class="text-base" />
                {{ $t("cloud.relink-instance") }}
              </a>
              <button v-else class="btn btn-sm flex-1" @click="fetchCloudStatus">
                <mdi:refresh class="text-base" />
                {{ $t("button.retry") }}
              </button>
            </div>
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

            <div class="px-2 py-1">
              <CloudUsage compact :usage="cloudStatus.usage" />
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
