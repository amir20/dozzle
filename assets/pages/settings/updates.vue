<template>
  <section class="flex flex-col gap-4">
    <h2 class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">{{ $t("settings.updates") }}</h2>
    <SelfUpdateStatus v-if="status && autoUpdate" :status="status" :auto-update="autoUpdate" />
    <!-- Off reads as a value, not a control: the schedule is still changed in setup. -->
    <div
      v-else-if="status?.autoUpdate"
      class="border-base-content/15 bg-base-200/40 flex items-start justify-between gap-3 rounded-lg border p-4"
    >
      <span class="min-w-0 flex-1">
        <span class="block text-sm font-medium">{{ $t("setup.update.auto-label") }}</span>
        <span class="text-base-content/60 mt-0.5 block text-xs">{{ $t("setup.update.auto-desc") }}</span>
      </span>
      <span class="text-base-content/60 shrink-0 font-mono text-sm">{{ $t("setup.restart.off") }}</span>
    </div>
  </section>
</template>

<script lang="ts" setup>
// The layout fetches the status. The panel is the same one About used to hold: what
// the schedule will do and an Update now button.
const { status } = useSetup();

const autoUpdate = computed(() => {
  const update = status.value?.autoUpdate;
  return update && update.mode !== "off" ? update : undefined;
});
</script>
