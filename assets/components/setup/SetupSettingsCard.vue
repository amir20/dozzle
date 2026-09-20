<template>
  <!-- Closed setup is the one state with nothing behind it, so it is not a button. -->
  <div
    v-if="state === 'closed'"
    class="card card-border bg-base-200/40 flex-row items-center gap-3 p-4 text-left opacity-80"
  >
    <span class="bg-base-content/10 text-base-content/70 shrink-0 rounded-full p-2">
      <mdi:lock-outline class="size-5" />
    </span>
    <span class="min-w-0 flex-1">
      <span class="block text-sm font-medium">{{ $t("setup.settings-closed-title") }}</span>
      <span class="text-base-content/60 block text-xs">{{ $t("setup.settings-closed-desc") }}</span>
    </span>
  </div>

  <button
    v-else
    type="button"
    class="card card-border bg-base-200/40 hover:bg-base-300 flex-row items-center gap-3 p-4 text-left transition-colors"
    @click="openWizard"
  >
    <span v-if="state === 'done'" class="bg-success/10 text-success shrink-0 rounded-full p-2">
      <mdi:check class="size-5" />
    </span>
    <span v-else class="bg-base-content/10 text-base-content/70 shrink-0 rounded-full p-2">
      <mdi:rocket-launch-outline class="size-5" />
    </span>
    <span class="min-w-0 flex-1">
      <span class="block text-sm font-medium">
        {{ state === "done" ? $t("setup.settings-done-title") : $t("setup.settings-title") }}
      </span>
      <span class="text-base-content/60 block text-xs">
        {{ state === "done" ? $t("setup.settings-done-desc") : $t("setup.settings-desc") }}
      </span>
    </span>
    <mdi:chevron-right class="size-4 shrink-0 opacity-40" />
  </button>
</template>

<script lang="ts" setup>
import type { SetupStatus } from "@/composable/setup/setup";

// Until the status lands the card reads as the invitation it has always been,
// which is also what an install whose setup API is unreachable keeps showing.
const { status } = defineProps<{ status: SetupStatus | null }>();

const { openWizard } = useSetup();

// "Set up Dozzle" only makes sense while there is something to set up. Once login
// is on the card reports rather than asks, and once the no-login window has closed
// the wizard cannot save anything, so it stops inviting the click.
const state = computed<"todo" | "done" | "closed">(() => {
  if (!status) return "todo";
  if (setupLoginConfigured(status)) return "done";
  return status.authProvider === "none" && !status.canWrite ? "closed" : "todo";
});
</script>
