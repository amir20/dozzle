<script lang="ts" setup>
const router = useRouter();
const route = useRoute();

const store = useContainerStore();
const { visibleContainers } = storeToRefs(store);

watch(visibleContainers, (newValue) => {
  if (newValue) {
    if (route.query.name) {
      const [container, _] = visibleContainers.value.filter((c) => c.name == route.query.name);
      if (container) {
        router.push({ name: "container-id", params: { id: container.id } });
      } else {
        console.error(`No containers found matching name=${route.query.name}. Redirecting to /`);
        router.push({ name: "index" });
      }
    } else {
      console.error(`Expection query parameter name to be set. Redirecting to /`);
      router.push({ name: "index" });
    }
  }
});
</script>
<template></template>
