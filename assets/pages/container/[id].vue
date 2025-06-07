<template>
  <Search />
  <ContainerLog :id show-title :scrollable="pinnedLogs.length > 0" v-if="currentContainer" />
  <div v-else-if="ready" class="hero bg-base-200 min-h-screen">
    <div class="hero-content text-center">
      <div class="max-w-md">
        <p class="py-6 text-2xl font-bold">{{ $t("error.container-not-found") }}</p>
      </div>
    </div>
  </div>
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

const redirectTrigger = ref(false);
watch(currentContainer, () => (redirectTrigger.value = false));

watchEffect(() => {
  if (redirectTrigger.value) return;
  if (automaticRedirect.value === "none") return;
  if (!currentContainer.value) return;
  if (currentContainer.value.state === "running") return;
  if (Date.now() - +currentContainer.value.finishedAt > 5 * 60 * 1000) return;

  const nextContainer = allContainers.value
    .filter((c) => c.startedAt > currentContainer.value.startedAt && c.name === currentContainer.value.name)
    .sort((a, b) => +a.created - +b.created)[0];

  if (!nextContainer) return;

  if (automaticRedirect.value === "delayed") {
    redirectTrigger.value = true;
    showToast(
      {
        title: t("alert.similar-container-found.title"),
        message: t("alert.similar-container-found.message", { containerId: nextContainer.id }),
        type: "info",
        action: {
          label: t("button.cancel"),
          handler: () => {
            showToast(
              {
                title: t("alert.redirected.title"),
                message: t("alert.redirected.message", { containerId: nextContainer.id }),
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
        message: t("alert.redirected.message", { containerId: nextContainer.id }),
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
