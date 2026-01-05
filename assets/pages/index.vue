<template>
  <PageWithLinks>
    <section>
      <div class="mb-4 flex items-center justify-between">
        <h2 class="text-lg font-semibold">{{ $t("label.host-count", { count: hosts.length }) }}</h2>
        <button @click="hostsCollapsed = !hostsCollapsed" class="btn btn-ghost btn-sm">
          <mdi:chevron-down :class="{ 'rotate-180': !hostsCollapsed }" class="transition-transform" />
        </button>
      </div>
      <Transition name="collapse">
        <HostList v-show="!hostsCollapsed" />
      </Transition>
    </section>

    <section>
      <div class="mb-2 flex items-center justify-between">
        <h2 class="text-lg font-semibold">
          {{ $t("label.container", { count: runningContainers.length }) }}
        </h2>
        <button @click="containersCollapsed = !containersCollapsed" class="btn btn-ghost btn-sm">
          <mdi:chevron-down :class="{ 'rotate-180': !containersCollapsed }" class="transition-transform" />
        </button>
      </div>
      <Transition name="collapse">
        <ContainerTable v-show="!containersCollapsed" :containers="runningContainers" />
      </Transition>
    </section>
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

const runningContainers = computed(() => containers.value.filter((c) => c.state === "running"));

// Persist collapse state in localStorage
const hostsCollapsed = useStorage("DOZZLE_HOSTS_COLLAPSED", false);
const containersCollapsed = useStorage("DOZZLE_CONTAINERS_COLLAPSED", false);

watchEffect(() => {
  if (ready.value) {
    setTitle(t("title.dashboard", { count: runningContainers.value.length }));
  }
});
</script>
<style scoped>
:deep(tr td) {
  padding-top: 1em;
  padding-bottom: 1em;
}

.collapse-enter-active,
.collapse-leave-active {
  transition: all 0.2s ease;
  overflow: hidden;
}

.collapse-enter-from,
.collapse-leave-to {
  opacity: 0;
  max-height: 0;
}
</style>
