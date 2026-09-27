<template>
  <Search />
  <ContainerLog :id show-title :scrollable="pinnedLogs.length > 0" v-if="currentContainer" />
  <NotFound v-else-if="ready" :title="$t('error.container-not-found')" :hint="$t('error.container-not-found-hint')">
    <template #icon><octicon:container-24 class="size-5" /></template>
  </NotFound>
</template>

<script lang="ts" setup>
import { type Container } from "@/models/Container";
const route = useRoute("/container/[id]");
const id = toRef(() => route.params.id);
const containerStore = useContainerStore();
const currentContainer = containerStore.currentContainer(id);
const { ready } = storeToRefs(containerStore);
const pinnedLogsStore = usePinnedLogsStore();
const { pinnedLogs } = storeToRefs(pinnedLogsStore);
const { containers: allContainers } = storeToRefs(containerStore) as unknown as { containers: Ref<Container[]> };
const { showToast } = useToast();
const { t } = useI18n();
const router = useRouter();

watchEffect(() => {
  if (ready.value) {
    if (currentContainer.value) {
      setTitle(currentContainer.value.name);
    } else {
      setTitle("Not Found");
    }
  }
});

// The store drops a container once its host's next list no longer carries it, and a
// recreate (update, compose up) removes the old container before starting the new one,
// so the list that brings the successor is the same one that drops the container on
// screen. The page holds on to the last one it saw so it can still find the successor.
// Only this one object is kept, and only while the page is mounted.
const lastSeen = shallowRef<Container>();
watch(
  currentContainer,
  (c) => {
    if (c) lastSeen.value = c;
  },
  { immediate: true },
);
const redirectFrom = computed(
  () => currentContainer.value ?? (lastSeen.value?.id === id.value ? lastSeen.value : undefined),
);

const redirectTrigger = ref(false);
watch(currentContainer, () => (redirectTrigger.value = false));

watchEffect(() => {
  if (redirectTrigger.value) return;
  if (automaticRedirect.value === "none") return;
  const from = redirectFrom.value;
  if (!from) return;
  if (from.state === "running") return;
  if (Date.now() - +from.finishedAt > 5 * 60 * 1000) return;

  const nextContainer = allContainers.value
    .filter((c) => c.startedAt > from.startedAt && c.name === from.name && c.host === from.host)
    .sort((a, b) => +a.created - +b.created)[0];

  if (!nextContainer) return;

  if (automaticRedirect.value === "delayed") {
    redirectTrigger.value = true;
    showToast(
      {
        title: t("alert.similar-container-found.title"),
        message: t("alert.similar-container-found.message", { containerId: escapeHtml(nextContainer.id) }),
        type: "info",
        action: {
          label: t("button.cancel"),
          handler: () => {
            showToast(
              {
                title: t("alert.redirected.title"),
                message: t("alert.redirected.message", { containerId: escapeHtml(nextContainer.id) }),
                type: "info",
              },
              { expire: 5000 },
            );
            router.push({ name: "/container/[id]", params: { id: nextContainer.id } });
          },
        },
      },
      { timed: 4000 },
    );
  } else {
    router.push({ name: "/container/[id]", params: { id: nextContainer.id } });
    showToast(
      {
        title: t("alert.redirected.title"),
        message: t("alert.redirected.message", { containerId: escapeHtml(nextContainer.id) }),
        type: "info",
      },
      { expire: 3000 },
    );
  }
});
</script>
<route lang="yaml">
meta:
  menu: host
</route>
