<template>
  <div>
    <h2 class="text-2xl font-bold">{{ $t("setup.actions.title") }}</h2>
    <p class="text-base-content/60 mt-1 text-sm">{{ $t("setup.actions.subtitle") }}</p>

    <!-- The example comes first: a crashed container and the button that fixes it
         says what "actions" means faster than a sentence does. Illustration only. -->
    <div class="border-base-content/15 bg-base-200/40 mt-6 rounded-lg border p-4" aria-hidden="true">
      <div class="flex items-center gap-3">
        <span class="bg-error size-2 shrink-0 rounded-full"></span>
        <div class="min-w-0 flex-1">
          <div class="truncate font-mono text-sm font-semibold">api</div>
          <div class="text-base-content/60 text-xs">{{ $t("setup.actions.example-status") }}</div>
        </div>
        <span class="btn btn-sm pointer-events-none">
          <mdi:restart class="size-4" />
          {{ $t("setup.actions.example-restart") }}
        </span>
      </div>
      <div class="bg-base-300 text-base-content/80 mt-3 truncate rounded-md px-2 py-1.5 font-mono text-xs">
        <span class="text-error">ERROR</span> connect ECONNREFUSED 10.0.0.4:5432
      </div>
    </div>

    <InlineNotice v-if="!status.canWrite" type="info" class="mt-4">
      {{ status.authProvider === "none" ? $t("setup.actions.window-closed") : $t("setup.actions.no-access") }}
    </InlineNotice>

    <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 mt-4 divide-y rounded-lg border">
      <label class="flex items-start justify-between gap-3 p-4">
        <span class="min-w-0 flex-1">
          <span class="block text-sm font-medium">{{ $t("setup.actions.actions-label") }}</span>
          <span class="text-base-content/60 mt-0.5 block text-xs">{{ $t("setup.actions.actions-desc") }}</span>
          <span v-if="status.locked.enableActions" class="text-base-content/40 mt-1 flex items-center gap-1 text-xs">
            <mdi:lock-outline class="size-3.5" />
            {{ $t("setup.actions.locked", { env: "DOZZLE_ENABLE_ACTIONS" }) }}
          </span>
        </span>
        <input
          v-model="enableActions"
          type="checkbox"
          class="toggle toggle-primary toggle-sm mt-0.5 shrink-0"
          :disabled="!canEdit('enableActions')"
        />
      </label>

      <label class="flex items-start justify-between gap-3 p-4">
        <span class="min-w-0 flex-1">
          <span class="block text-sm font-medium">{{ $t("setup.actions.shell-label") }}</span>
          <span class="text-base-content/60 mt-0.5 block text-xs">{{ $t("setup.actions.shell-desc") }}</span>
          <span v-if="status.locked.enableShell" class="text-base-content/40 mt-1 flex items-center gap-1 text-xs">
            <mdi:lock-outline class="size-3.5" />
            {{ $t("setup.actions.locked", { env: "DOZZLE_ENABLE_SHELL" }) }}
          </span>
        </span>
        <input
          v-model="enableShell"
          type="checkbox"
          class="toggle toggle-primary toggle-sm mt-0.5 shrink-0"
          :disabled="!canEdit('enableShell')"
        />
      </label>

      <div class="flex items-start gap-3 p-4">
        <div class="bg-warning/10 text-warning shrink-0 rounded-full p-1.5">
          <mdi:console class="size-4" />
        </div>
        <p class="text-base-content/60 text-xs leading-relaxed">{{ $t("setup.actions.shell-warning") }}</p>
      </div>
    </div>

    <p class="text-base-content/40 mt-3 text-xs">{{ $t("setup.actions.restart-note") }}</p>

    <InlineNotice v-if="error" type="error" class="mt-4">{{ error }}</InlineNotice>
  </div>
</template>

<script lang="ts" setup>
import type { SetupNextResult, SetupStatus } from "@/composable/setup/setup";

const { status } = defineProps<{ status: SetupStatus }>();

const { t } = useI18n();
const { saveConfig } = useSetup();

type Field = "enableActions" | "enableShell";

const initial = setupToggles(status);
const enableActions = ref(initial.enableActions);
const enableShell = ref(initial.enableShell);
const saving = ref(false);
const error = ref("");

function canEdit(field: Field) {
  return status.canWrite && !status.locked[field];
}

async function next(): Promise<SetupNextResult> {
  error.value = "";
  const current = setupToggles(status);
  const patch: Partial<Record<Field, boolean>> = {};
  if (canEdit("enableActions") && enableActions.value !== current.enableActions) {
    patch.enableActions = enableActions.value;
  }
  if (canEdit("enableShell") && enableShell.value !== current.enableShell) {
    patch.enableShell = enableShell.value;
  }
  if (Object.keys(patch).length === 0) return "advance";

  saving.value = true;
  try {
    await saveConfig(patch);
    return "advance";
  } catch {
    error.value = t("setup.error.generic");
    return "stay";
  } finally {
    saving.value = false;
  }
}

const nextLabel = computed(() => t("setup.next"));
const nextDisabled = computed(() => saving.value);

defineExpose({ nextLabel, nextDisabled, nextPlain: false, busy: saving, next });
</script>
