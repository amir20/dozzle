<script lang="ts" setup>
const router = useRouter();
const route = useRoute();

const store = useContainerStore();
const { containers } = storeToRefs(store);

watch(containers, (newValue) => {
  if (newValue) {
    if (route.query.name) {
      const name = route.query.name as string;
      const host = route.query.host as string | undefined;
      const matches = containers.value
        .filter((c) => c.name == name && (!host || c.host == host))
        .sort((a, b) => b.startedAt.getTime() - a.startedAt.getTime());
      if (matches.length > 0) {
        router.push({ name: "/container/[id]", params: { id: matches[0].id } });
      } else {
        console.error(`No containers found matching name=${name}${host ? ` host=${host}` : ""}. Redirecting to /`);
        router.push({ name: "/" });
      }
    } else {
      console.error(`Expection query parameter name to be set. Redirecting to /`);
      router.push({ name: "/" });
    }
  }
});
</script>
<template></template>
