<template>
  <!-- What the next restart changes. Rows on the wizard's last step, one line in the
       settings banner. -->
  <p v-if="compact" class="text-base-content/60 text-xs">
    <template v-for="(row, i) in rows" :key="row.key">
      <span v-if="i > 0"> · </span>
      {{ row.label }} <span class="font-mono font-semibold">{{ row.value }}</span>
    </template>
  </p>
  <div v-else class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
    <div v-for="row in rows" :key="row.key" class="flex items-center justify-between gap-3 p-4">
      <span class="text-sm">{{ row.label }}</span>
      <span class="font-mono text-xs font-semibold">{{ row.value }}</span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { SetupStatus } from "@/composable/setup/setup";

const { status, compact = false } = defineProps<{ status: SetupStatus; compact?: boolean }>();

const { t } = useI18n();

const labels = {
  auth: "setup.restart.change-auth",
  actions: "setup.actions.actions-label",
  shell: "setup.actions.shell-label",
  update: "setup.steps.update",
} as const;

const rows = computed(() =>
  setupPendingChanges(status).map((change) => {
    let value: string;
    if (typeof change.value === "boolean") value = change.value ? t("setup.restart.on") : t("setup.restart.off");
    else if (change.key === "update")
      value = `${change.value === "daily" ? t("setup.update.daily") : t("setup.update.weekly")} · ${change.time}`;
    else value = change.value;
    return { key: change.key, label: t(labels[change.key]), value };
  }),
);
</script>
