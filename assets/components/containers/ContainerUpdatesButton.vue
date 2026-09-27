<template>
  <!-- Quiet until there is something to do: in automatic mode it only appears
       once a check found an update, so a dashboard that is current looks
       exactly as it did before. Manual mode never reaches a registry on its
       own, so it offers the check instead. -->
  <button
    v-if="count > 0 || running"
    type="button"
    class="btn btn-xs btn-ghost gap-1.5"
    :aria-label="$t('updates.count', count)"
    @click="open"
  >
    <span class="bg-info/10 text-info flex size-4 items-center justify-center rounded-full">
      <span v-if="running" class="loading loading-spinner size-2.5"></span>
      <mdi:arrow-up class="size-3" v-else />
    </span>
    <span class="font-mono">{{ running ? $t("updates.running") : $t("updates.count", count) }}</span>
  </button>
  <button
    v-else-if="config.imageCheckMode === 'manual'"
    type="button"
    class="btn btn-xs btn-ghost gap-1.5"
    :disabled="checking"
    @click="checkAll(true)"
  >
    <span v-if="checking" class="loading loading-spinner size-3"></span>
    <mdi:update v-else class="size-3.5 opacity-60" />
    {{ checking ? $t("toolbar.checking-for-updates") : $t("toolbar.check-for-updates") }}
  </button>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import ContainerUpdatesDrawer from "./ContainerUpdatesDrawer.vue";

// Every container, not just the rows the dashboard filter shows, so the count
// matches what the drawer lists.
const { containers } = storeToRefs(useContainerStore()) as unknown as { containers: Ref<Container[]> };
const { checkAll, checking, hasUpdate } = useImageUpdates();
const { running, hold } = useBulkUpdate();
const showDrawer = useDrawer();

const count = computed(() => containers.value.filter((c) => c.state !== "deleted" && hasUpdate(c)).length);

onMounted(() => checkAll());

// Watched for as long as the dashboard is up, so a job started by the schedule
// or another tab shows here and the drawer can be opened to follow it.
onScopeDispose(hold());

function open() {
  showDrawer(ContainerUpdatesDrawer, {}, "lg");
}
</script>
