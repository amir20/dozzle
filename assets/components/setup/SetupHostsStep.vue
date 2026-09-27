<template>
  <div>
    <h2 class="text-2xl font-bold">{{ $t("setup.hosts.title") }}</h2>
    <p class="text-base-content/60 mt-1 text-sm">{{ $t("setup.hosts.subtitle") }}</p>

    <AddHostPanel ref="panel" :status="status" class="mt-6" />
  </div>
</template>

<script lang="ts" setup>
import type { SetupNextResult, SetupStatus } from "@/composable/setup/setup";
import type AddHostPanel from "@/components/hosts/AddHostPanel.vue";

const { status } = defineProps<{ status: SetupStatus }>();

const { t } = useI18n();
const panel = useTemplateRef<InstanceType<typeof AddHostPanel>>("panel");

const hasAgents = computed(() => (status.agents?.length ?? 0) > 0);

// A host is live the moment it is added, so there is nothing left for Next to save.
// With none added, Next only passes the step over, the same as Skip.
async function next(): Promise<SetupNextResult> {
  return hasAgents.value ? "advance" : "skip";
}

const busy = computed(() => !!panel.value?.busy);
const nextLabel = computed(() => t("setup.next"));
const skipLabel = computed(() => (hasAgents.value ? undefined : t("setup.hosts.skip")));

// Optional, like the Cloud step: the footer's Next stays plain so the panel's own
// "Add host" is the one primary button on screen.
defineExpose({ nextLabel, nextDisabled: busy, nextPlain: true, skipLabel, busy, next });
</script>
