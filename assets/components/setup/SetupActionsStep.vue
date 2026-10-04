<template>
  <div>
    <h2 class="text-2xl font-bold">{{ $t("setup.actions.title") }}</h2>
    <p class="text-base-content/60 mt-1 text-sm">{{ $t("setup.actions.subtitle") }}</p>

    <!-- The example comes first: a crashed container and the button that fixes it
         says what "actions" means faster than a sentence does. Illustration only. -->
    <div class="border-base-content/15 bg-base-200/40 mt-6 rounded-lg border p-4" aria-hidden="true">
      <span class="status-pill status-pill-neutral mb-3">{{ $t("setup.actions.example") }}</span>
      <div class="flex items-center gap-3">
        <span class="bg-error size-2 shrink-0 rounded-full"></span>
        <div class="min-w-0 flex-1">
          <div class="truncate font-mono text-sm font-semibold">api</div>
          <div class="text-base-content/60 text-xs">{{ $t("setup.actions.example-status") }}</div>
        </div>
        <!-- Drawn, not a button: a real-looking button here gets clicked and does nothing. -->
        <span
          class="border-base-content/15 text-base-content/60 inline-flex items-center gap-1.5 rounded-md border border-dashed px-2.5 py-1 text-xs"
        >
          <mdi:restart class="size-3.5" />
          {{ $t("setup.actions.example-restart") }}
        </span>
      </div>
      <div class="bg-base-300 text-base-content/80 mt-3 truncate rounded-md px-2 py-1.5 font-mono text-xs">
        <span class="text-error">ERROR</span> connect ECONNREFUSED 10.0.0.4:5432
      </div>
    </div>

    <!-- The same switches as Settings → Security, saved on Next instead of on change. -->
    <SetupTogglesForm ref="form" :status="status" class="mt-4" />

    <p class="text-base-content/40 mt-3 text-xs">{{ $t("setup.actions.restart-note") }}</p>
  </div>
</template>

<script lang="ts" setup>
import type { SetupNextResult, SetupStatus } from "@/composable/setup/setup";
import SetupTogglesForm from "./SetupTogglesForm.vue";

defineProps<{ status: SetupStatus }>();

const { t } = useI18n();
const form = useTemplateRef<InstanceType<typeof SetupTogglesForm>>("form");

async function next(): Promise<SetupNextResult> {
  return (await form.value?.save()) === false ? "stay" : "advance";
}

const saving = computed(() => !!form.value?.saving);
const dirty = computed(() => !!form.value?.dirty);
const actionsDraft = computed(() => form.value?.actionsDraft);
const nextLabel = computed(() => t("setup.next"));

defineExpose({ nextLabel, nextDisabled: saving, nextPlain: false, actionsDraft, dirty, busy: saving, next });
</script>
