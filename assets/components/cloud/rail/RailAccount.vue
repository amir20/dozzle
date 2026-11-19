<template>
  <!--
    The account and usage, opened from the cloud mark. Every branch is the same
    panel, so nothing reshuffles as the status loads or the connection drops.
  -->
  <div class="h-full overflow-y-auto p-4">
    <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
      <!-- Severity rides on the icon; the panel stays neutral. -->
      <div v-if="cloudStatusError" class="flex items-start gap-3 p-4" data-error>
        <span
          class="shrink-0 rounded-full p-1.5"
          :class="cloudStatusError === 'auth' ? 'bg-error/10 text-error' : 'bg-warning/10 text-warning'"
        >
          <mdi:alert-circle-outline v-if="cloudStatusError === 'auth'" class="size-5" />
          <mdi:cloud-off-outline v-else class="size-5" />
        </span>
        <div class="min-w-0 flex-1">
          <p class="text-sm">
            {{ cloudStatusError === "auth" ? $t("cloud.error") : $t("cloud.error-unavailable") }}
          </p>
          <button
            v-if="cloudStatusError === 'auth' && canLink"
            type="button"
            class="btn btn-primary btn-sm mt-3"
            @click="startCloudLink('cloud')"
          >
            <mdi:link-variant class="text-base" />
            {{ $t("cloud.relink-instance") }}
          </button>
          <button v-else type="button" class="btn btn-sm mt-3" @click="fetchCloudStatus">
            <mdi:refresh class="text-base" />
            {{ $t("button.retry") }}
          </button>
        </div>
      </div>

      <div v-else-if="!cloudStatus" class="flex items-center justify-center p-6">
        <span class="loading loading-spinner loading-xs"></span>
      </div>

      <template v-else>
        <div class="flex items-center justify-between gap-3 p-4 text-sm">
          <span class="text-base-content/60">{{ $t("cloud.account") }}</span>
          <span class="min-w-0 truncate font-mono" data-email>{{ cloudStatus.user.email }}</span>
        </div>
        <div class="flex items-center justify-between gap-3 p-4 text-sm">
          <span class="text-base-content/60">{{ $t("cloud.plan") }}</span>
          <span class="status-pill status-pill-primary capitalize">{{ cloudStatus.plan.name }}</span>
        </div>
        <div class="p-4">
          <CloudUsage :usage="cloudStatus.usage" />
        </div>
        <div class="p-2">
          <a
            :href="dashboardHref"
            target="_blank"
            rel="noreferrer noopener"
            class="hover:bg-base-300 flex items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors"
          >
            <mdi:view-dashboard-outline class="size-4 opacity-60" />
            <span class="flex-1">{{ $t("cloud.dashboard") }}</span>
            <mdi:open-in-new class="size-3.5 opacity-40" />
          </a>
        </div>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
const { canLink } = useCloudSurface();
const { cloudStatus, cloudStatusError, fetchCloudStatus } = useCloudConfig();

const dashboardHref = config.cloudUrl;

// Fresh on every open: usage is the thing people come here to read.
onMounted(fetchCloudStatus);
</script>
