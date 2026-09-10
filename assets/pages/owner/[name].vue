<template>
  <Search />
  <OwnerLog :owner="owner" :scrollable="pinnedLogs.length > 0" v-if="owner" />
</template>

<script lang="ts" setup>
const route = useRoute("/owner/[name]");

const containerStore = useContainerStore();
const { ready } = storeToRefs(containerStore);

const pinnedLogsStore = usePinnedLogsStore();
const { pinnedLogs } = storeToRefs(pinnedLogsStore);

const k8sStore = useK8sStore();
const { owners } = storeToRefs(k8sStore);
const ownerKey = computed(() => {
  try {
    return decodeURIComponent(String(route.params.name));
  } catch {
    return String(route.params.name);
  }
});
const owner = computed(() => owners.value.find((o) => o.key === ownerKey.value));

watchEffect(() => {
  if (ready.value) {
    if (owner.value?.name) {
      setTitle(`${owner.value.kind}/${owner.value.name}`);
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
