<template>
  <div class="border-base-content/15 bg-base-200/40 flex items-start gap-3 rounded-lg border p-4">
    <div v-if="timedOut" class="bg-warning/10 text-warning shrink-0 rounded-full p-2">
      <mdi:timer-sand class="size-5" />
    </div>
    <div v-else class="bg-info/10 text-info flex shrink-0 rounded-full p-2">
      <span class="loading loading-spinner loading-sm"></span>
    </div>
    <div class="min-w-0 flex-1">
      <div class="text-sm font-semibold">
        {{ timedOut ? $t("setup.restart.slow-title") : $t("setup.restart.restarting") }}
      </div>
      <p class="text-base-content/60 mt-0.5 text-sm">
        {{ timedOut ? $t("setup.restart.slow-body") : hint }}
      </p>
      <button v-if="timedOut" type="button" class="btn btn-sm mt-3" @click="reload">
        <mdi:refresh class="size-4" />
        {{ $t("setup.restart.reload") }}
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
defineProps<{ timedOut: boolean; hint?: string }>();

function reload() {
  window.location.assign(withBase("/"));
}
</script>
