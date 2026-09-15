<script lang="ts" setup>
const router = useRouter();
const route = useRoute();

const store = useContainerStore();
const { containers, ready } = storeToRefs(store);

whenever(
  ready,
  () => {
    const names = String(route.query.name ?? "")
      .split(",")
      .map((n) => n.trim())
      .filter(Boolean);
    if (names.length === 0) {
      console.error(`Expected query parameter name to be set. Redirecting to /`);
      router.replace({ name: "/" });
      return;
    }

    const host = route.query.host as string | undefined;
    const ids: string[] = [];
    for (const name of names) {
      const match = containers.value
        .filter((c) => c.name == name && (!host || c.host == host))
        .sort((a, b) => b.startedAt.getTime() - a.startedAt.getTime())[0];
      if (match) {
        if (!ids.includes(match.id)) ids.push(match.id);
      } else {
        console.warn(`No containers found matching name=${name}${host ? ` host=${host}` : ""}`);
      }
    }

    if (ids.length === 1) {
      router.replace({ name: "/container/[id]", params: { id: ids[0] } });
    } else if (ids.length > 1) {
      router.replace({ name: "/merged/[ids]", params: { ids: ids.join(",") } });
    } else {
      console.error(`No containers found. Redirecting to /`);
      router.replace({ name: "/" });
    }
  },
  { immediate: true, once: true },
);
</script>
<template></template>
