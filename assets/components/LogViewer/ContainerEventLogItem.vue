<template>
  <div class="flex w-full flex-col">
    <div class="relative flex items-start gap-x-2">
      <ContainerName class="flex-none" :id="logEntry.containerID" v-if="showContainerName" />
      <LogDate :date="logEntry.date" v-if="showTimestamp" />
      <LogLevel class="flex" />
      <div class="whitespace-pre-wrap" :data-event="logEntry.event" v-html="logEntry.message"></div>
    </div>
    <div class="alert alert-info mt-8 w-auto flex-none font-sans text-[1rem] md:mx-auto md:w-1/2" v-if="followEligible">
      <carbon:information class="size-6 shrink-0 stroke-current" />
      <div>
        <h3 class="text-lg font-bold">{{ $t("alert.similar-container-found.title") }}</h3>
        {{ $t("alert.similar-container-found.message", { containerId: nextContainer.id }) }}
      </div>
      <div>
        <TimedButton v-if="automaticRedirect" class="btn-primary btn-sm" @finished="redirectNow()">{{
          $t("button.cancel")
        }}</TimedButton>
        <router-link
          :to="{ name: '/container/[id]', params: { id: nextContainer.id } }"
          class="btn btn-primary btn-sm"
          v-else
        >
          {{ $t("button.redirect") }}
        </router-link>
      </div>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { ContainerEventLogEntry } from "@/models/LogEntry";
const router = useRouter();
const { showToast } = useToast();
const { t } = useI18n();

const { logEntry } = defineProps<{
  logEntry: ContainerEventLogEntry;
  showContainerName?: boolean;
}>();

const { containers } = useLoggingContext();

const store = useContainerStore();

const { containers: allContainers } = storeToRefs(store);

const nextContainer = computed(
  () =>
    [
      ...allContainers.value.filter(
        (c) =>
          c.host === containers.value[0].host &&
          c.created > logEntry.date &&
          c.name === containers.value[0].name &&
          c.state === "running",
      ),
    ].sort((a, b) => +a.created - +b.created)[0],
);

const followEligible = computed(
  () =>
    router.currentRoute.value.name === "/container/[id]" &&
    logEntry.event === "container-stopped" &&
    containers.value.length === 1 &&
    nextContainer.value !== undefined,
);

function redirectNow() {
  showToast(
    {
      title: t("alert.redirected.title"),
      message: t("alert.redirected.message", { containerId: nextContainer.value?.id }),
      type: "info",
    },
    { expire: 5000 },
  );
  router.push({ name: "/container/[id]", params: { id: nextContainer.value?.id } });
}
</script>

<style lang="postcss" scoped>
[data-event="container-stopped"] {
  @apply text-red;
}
[data-event="container-started"] {
  @apply text-green;
}
</style>
