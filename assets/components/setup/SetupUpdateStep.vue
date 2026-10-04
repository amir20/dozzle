<template>
  <div>
    <span class="status-pill status-pill-success">
      {{ status.enableActions ? $t("setup.update.pill-on") : $t("setup.update.pill-pending") }}
    </span>
    <h2 class="mt-3 text-2xl font-bold">{{ $t("setup.update.title") }}</h2>
    <p class="text-base-content/60 mt-1 text-sm">{{ $t("setup.update.subtitle") }}</p>

    <!-- The same form as Settings → Updates, saved on Next instead of on change. Hidden
         while Dozzle replaces itself, so the restart is all the step shows. -->
    <SetupAutoUpdateForm v-if="!restarting" ref="form" :status="status" class="mt-6" />

    <!-- Coming back from the restart lands on this step again, showing the version it now runs. -->
    <SetupSelfUpdateStatus
      ref="updater"
      :status="status"
      :auto-update="autoUpdate"
      variant="version"
      resume="update"
      :class="restarting ? 'mt-6' : 'mt-4'"
    />

    <p v-if="!status.enableActions && !restarting" class="text-base-content/40 mt-3 text-xs">
      {{ $t("setup.update.needs-actions") }}
    </p>
  </div>
</template>

<script lang="ts" setup>
import type { SetupNextResult, SetupStatus } from "@/composable/setup/setup";
import SetupAutoUpdateForm from "./SetupAutoUpdateForm.vue";
import SetupSelfUpdateStatus from "./SetupSelfUpdateStatus.vue";

const { status } = defineProps<{ status: SetupStatus }>();

const { t } = useI18n();

const form = useTemplateRef<InstanceType<typeof SetupAutoUpdateForm>>("form");
const updater = useTemplateRef<InstanceType<typeof SetupSelfUpdateStatus>>("updater");

// The wizard only mounts this step when the server reports autoUpdate.
const autoUpdate = computed(() => setupAutoUpdate(status));

const phase = computed(() => updater.value?.phase ?? "idle");
const restarting = computed(() => phase.value === "restarting" || phase.value === "timeout");

async function next(): Promise<SetupNextResult> {
  return (await form.value?.save()) === false ? "stay" : "advance";
}

const saving = computed(() => !!form.value?.saving);
const nextLabel = computed(() => t("setup.next"));
const nextDisabled = computed(() => saving.value || phase.value === "pulling" || phase.value === "restarting");
const busy = computed(() => saving.value || phase.value === "pulling" || phase.value === "restarting");
// No separate skip: the form is the choice, and leaving the schedule off then Next
// already declines. A "Not now" beside Next would do exactly the same thing.
const dirty = computed(() => !!form.value?.dirty);

defineExpose({ nextLabel, nextDisabled, nextPlain: false, dirty, busy, next });
</script>
