<template>
  <Search />
  <SideDrawer ref="drawer">
    <LogDetails :entry="entry" />
  </SideDrawer>
  <ContainerLog :id="id" :show-title="true" :scrollable="pinnedLogs.length > 0" v-if="currentContainer" />
  <div v-else-if="ready" class="hero min-h-screen bg-base-200">
    <div class="hero-content text-center">
      <div class="max-w-md">
        <p class="py-6 text-2xl font-bold">{{ $t("error.container-not-found") }}</p>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import SideDrawer from "@/components/common/SideDrawer.vue";
import { JSONObject, LogEntry } from "@/models/LogEntry";
const route = useRoute("/container/[id]");
const id = toRef(() => route.params.id);

const containerStore = useContainerStore();
const currentContainer = containerStore.currentContainer(id);
const { ready } = storeToRefs(containerStore);

const pinnedLogsStore = usePinnedLogsStore();
const { pinnedLogs } = storeToRefs(pinnedLogsStore);

const drawer = ref<InstanceType<typeof SideDrawer>>();
const entry = ref<LogEntry<string | JSONObject>>();

export const showLogDetails = Symbol("showLogDetails") as InjectionKey<(l: LogEntry<string | JSONObject>) => void>;

provide(showLogDetails, (logEntry: LogEntry<string | JSONObject>) => {
  entry.value = logEntry;
  drawer.value?.open();
});

watchEffect(() => {
  if (ready.value) {
    if (currentContainer.value) {
      setTitle(currentContainer.value.name);
    } else {
      setTitle("Not Found");
    }
  }
});
</script>
<route lang="yaml">
meta:
  containerMode: true
</route>
