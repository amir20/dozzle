<template>
  <Search />
  <NamespaceLog :namespace="namespace" :scrollable="pinnedLogs.length > 0" v-if="namespace" />
</template>

<script lang="ts" setup>
const route = useRoute("/namespace/[name]");

const containerStore = useContainerStore();
const { ready } = storeToRefs(containerStore);

const pinnedLogsStore = usePinnedLogsStore();
const { pinnedLogs } = storeToRefs(pinnedLogsStore);

const k8sStore = useK8sStore();
const { namespaces } = storeToRefs(k8sStore);
const namespace = computed(() => namespaces.value.find((ns) => ns.name === route.params.name));

watchEffect(() => {
  if (ready.value) {
    if (namespace.value?.name) {
      setTitle(namespace.value.name);
    } else {
      setTitle("Not Found");
    }
  }
});
</script>
<route lang="yaml">
meta:
  menu: k8s
</route>
