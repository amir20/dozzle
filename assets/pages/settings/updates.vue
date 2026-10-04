<template>
  <!-- Dozzle's own image: what the schedule will do to it, Update now, and the schedule
       itself. The same form as the wizard's Updates step, saved as it changes. -->
  <section class="flex flex-col gap-4">
    <div>
      <h2 class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">{{ $t("settings.updates") }}</h2>
      <p class="text-base-content/60 mt-1 text-sm">{{ $t("setup.update.subtitle") }}</p>
    </div>

    <InlineNotice v-if="status && !status.enableActions" type="info">
      {{ $t("setup.update.needs-actions") }}
      <template #actions>
        <router-link to="/settings/security" class="btn btn-sm">{{ $t("settings.security") }}</router-link>
      </template>
    </InlineNotice>

    <SelfUpdateStatus v-if="status && scheduled" :status="status" :auto-update="scheduled" />

    <AutoUpdateForm v-if="status" :status="status" autosave />
    <SetupStatusMissing v-else :loading="loading" />
  </section>
</template>

<script lang="ts" setup>
// The layout fetches the status. This page only shows in server mode.
const { status, loading } = useSetup();

const scheduled = computed(() => {
  const update = status.value?.autoUpdate;
  return update && update.mode !== "off" ? update : undefined;
});
</script>
