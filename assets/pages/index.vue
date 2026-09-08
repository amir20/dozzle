<template>
  <PageWithLinks>
    <CollapsibleSection v-model="hostsCollapsed" :title="$t('label.hosts')" :count="Object.keys(hosts).length">
      <HostList />
    </CollapsibleSection>

    <CollapsibleSection
      v-model="containersCollapsed"
      :title="$t('label.containers')"
      :count="dashboardContainers.length"
    >
      <template #actions>
        <div class="join max-md:hidden">
          <button
            class="icon-btn btn join-item btn-xs"
            :class="statMode === 'chart' ? 'btn-active' : 'btn-ghost'"
            :aria-pressed="statMode === 'chart'"
            @click="statMode = 'chart'"
          >
            <mdi:chart-bar />
          </button>
          <button
            class="icon-btn btn join-item btn-xs"
            :class="statMode === 'progress' ? 'btn-active' : 'btn-ghost'"
            :aria-pressed="statMode === 'progress'"
            @click="statMode = 'progress'"
          >
            <mdi:poll class="scale-x-[-1] rotate-90" />
          </button>
        </div>
      </template>
      <ContainerTable :containers="dashboardContainers" v-model:stat-mode="statMode" />
    </CollapsibleSection>
  </PageWithLinks>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";

const { t } = useI18n();
const { hosts } = useHosts();

const containerStore = useContainerStore();
const { containers, ready } = storeToRefs(containerStore) as unknown as {
  containers: Ref<Container[]>;
  ready: Ref<boolean>;
};

const dashboardContainers = computed(() =>
  containers.value.filter((c) => (showAllContainers.value ? c.state !== "deleted" : c.state === "running")),
);

const statMode = useStorage<"chart" | "progress">("DOZZLE_TABLE_STAT_MODE", "chart");
const hostsCollapsed = useStorage("DOZZLE_HOSTS_COLLAPSED", false);
const containersCollapsed = useStorage("DOZZLE_CONTAINERS_COLLAPSED", false);

watchEffect(() => {
  if (ready.value) {
    setTitle(t("title.dashboard", { count: dashboardContainers.value.length }));
  }
});
</script>
