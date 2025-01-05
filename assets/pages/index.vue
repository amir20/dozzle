<template>
  <PageWithLinks>
    <section>
      <HostList />
    </section>

    <section>
      <ContainerTable :containers="runningContainers"></ContainerTable>
    </section>
  </PageWithLinks>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";

const { t } = useI18n();

const containerStore = useContainerStore();
const { containers, ready } = storeToRefs(containerStore) as unknown as {
  containers: Ref<Container[]>;
  ready: Ref<boolean>;
};

const runningContainers = computed(() => containers.value.filter((c) => c.state === "running"));

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
</style>
