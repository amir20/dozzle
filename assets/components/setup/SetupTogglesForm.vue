<template>
  <!-- Container actions and shell: the same two switches in the setup wizard and on
       Settings → Security. The wizard saves on Next; Settings saves as you change them. -->
  <div class="flex flex-col gap-4">
    <SetupAccessNotice v-if="notices" :status="status" />

    <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
      <label v-for="field in fields" :key="field.id" class="flex items-start justify-between gap-3 p-4">
        <span class="min-w-0 flex-1">
          <span class="flex flex-wrap items-center gap-2 text-sm font-medium">
            {{ $t(field.label) }}
            <!-- Saved, not running yet: the banner above offers the restart. -->
            <span v-if="autosave && status.pending[field.id] != null" class="status-pill status-pill-warning">
              {{ $t("settings.after-restart") }}
            </span>
          </span>
          <span class="text-base-content/60 mt-0.5 block text-xs">{{ $t(field.desc) }}</span>
          <SetupLocked v-if="status.locked[field.id]" :env="field.env" class="mt-1" />
        </span>
        <input
          v-if="canEdit(field.id)"
          v-model="draft[field.id]"
          type="checkbox"
          class="toggle toggle-primary toggle-sm mt-0.5 shrink-0"
          :disabled="saving"
        />
        <!-- Read only, so text. -->
        <span v-else class="shrink-0 text-sm" :class="{ 'text-base-content/60': !draft[field.id] }">
          {{ draft[field.id] ? $t("setup.restart.on") : $t("setup.restart.off") }}
        </span>
      </label>

      <div class="flex items-start gap-3 p-4">
        <div class="bg-warning/10 text-warning shrink-0 rounded-full p-1.5">
          <mdi:console class="size-4" />
        </div>
        <p class="text-base-content/60 text-xs leading-relaxed">{{ $t("setup.actions.shell-warning") }}</p>
      </div>
    </div>

    <InlineNotice v-if="error" type="error">{{ error }}</InlineNotice>
  </div>
</template>

<script lang="ts" setup>
import type { SetupStatus } from "@/composable/setup/setup";

const {
  status,
  autosave = false,
  notices = true,
} = defineProps<{
  status: SetupStatus;
  // False when the page already shows the access notice for every form on it.
  notices?: boolean;
  // Save every change right away, as the rest of Settings does. The wizard saves on Next.
  autosave?: boolean;
}>();

const { t } = useI18n();
const { saveConfig } = useSetup();

type Field = "enableActions" | "enableShell";

const fields = [
  {
    id: "enableActions",
    label: "setup.actions.actions-label",
    desc: "setup.actions.actions-desc",
    env: "DOZZLE_ENABLE_ACTIONS",
  },
  {
    id: "enableShell",
    label: "setup.actions.shell-label",
    desc: "setup.actions.shell-desc",
    env: "DOZZLE_ENABLE_SHELL",
  },
] as const satisfies readonly { id: Field; label: string; desc: string; env: string }[];

// What each switch will be after the next restart, until someone changes it here.
const draft = reactive<Record<Field, boolean>>({ ...setupToggles(status) });
const saving = ref(false);
const error = ref("");

function canEdit(field: Field) {
  return setupCanEdit(status) && !status.locked[field];
}

function changes() {
  const current = setupToggles(status);
  const patch: Partial<Record<Field, boolean>> = {};
  for (const { id } of fields) if (canEdit(id) && draft[id] !== current[id]) patch[id] = draft[id];
  return patch;
}

const dirty = computed(() => Object.keys(changes()).length > 0);

async function save(): Promise<boolean> {
  const patch = changes();
  if (Object.keys(patch).length === 0) return true;
  error.value = "";
  saving.value = true;
  try {
    await saveConfig(patch);
    return true;
  } catch {
    error.value = t("setup.error.generic");
    // A switch never shows a value that is not saved.
    if (autosave) Object.assign(draft, setupToggles(status));
    return false;
  } finally {
    saving.value = false;
  }
}

if (autosave) watch(draft, () => save());

// What the server says now, unless something here is still waiting to be saved.
watch(
  () => setupToggles(status),
  (saved) => {
    if (saving.value || dirty.value) return;
    Object.assign(draft, saved);
  },
);

// The actions switch's unsaved value, so the wizard greys Auto-update in or out right away.
const actionsDraft = computed(() => draft.enableActions);

defineExpose({ dirty, save, saving, actionsDraft });
</script>
