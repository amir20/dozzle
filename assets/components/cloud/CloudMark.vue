<template>
  <!-- The cloud mark at the top of the rail (or of the phone's cloud screen). Its
       colour and dot say whether the connection is healthy; it opens the account. -->
  <button
    type="button"
    class="icon-btn btn btn-ghost btn-square btn-sm relative"
    :class="{ 'bg-base-300': active }"
    :title="$t('cloud.title')"
    :aria-label="$t('cloud.title')"
    :aria-pressed="active"
    @click="isMobile ? openRail('account') : toggleRail('account')"
  >
    <mdi:cloud
      class="icon-float size-6"
      :class="!cloudStatusError ? 'text-info' : cloudStatusError === 'unavailable' ? 'text-warning' : 'text-error'"
    />
    <span
      class="absolute top-0.5 right-0.5 size-2 rounded-full"
      :class="!cloudStatusError ? 'bg-success' : cloudStatusError === 'unavailable' ? 'bg-warning' : 'bg-error'"
    ></span>
  </button>
</template>

<script lang="ts" setup>
const { panel, openRail, toggleRail } = useCloudRail();
const { cloudStatusError, ensureCloudStatus } = useCloudConfig();
const active = computed(() => panel.value === "account");

onMounted(ensureCloudStatus);
</script>
