<template>
  <div class="relative flex w-full items-start gap-x-2">
    <ContainerName class="flex-none" :id="logEntry.containerID" v-if="showContainerName" />
    <LogDate :date="logEntry.date" v-if="showTimestamp" />
    <LogLevel class="flex" />
    <div class="whitespace-pre-wrap" :data-event="logEntry.event" v-html="logEntry.message"></div>
  </div>
  <div
    class="alert alert-info mt-8 w-auto text-[1rem] md:mx-auto md:w-1/2"
    v-if="nextContainer && logEntry.event === 'container-stopped' && containers.length == 1"
  >
    <carbon:information class="size-6 shrink-0 stroke-current" />
    <div>
      <h3 class="text-lg font-bold">{{ $t("alert.similar-container-found.title") }}</h3>
      {{ $t("alert.similar-container-found.message", { containerId: nextContainer.id }) }}
    </div>
    <div>
      <TimedButton v-if="automaticRedirect" class="btn-primary btn-sm" @finished="redirectNow()">Cancel</TimedButton>
      <router-link
        :to="{ name: 'container-id', params: { id: nextContainer.id } }"
        class="btn btn-primary btn-sm"
        v-else
      >
        Redirect
      </router-link>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { DockerEventLogEntry } from "@/models/LogEntry";
const router = useRouter();
const { showToast } = useToast();
const { t } = useI18n();

const { logEntry } = defineProps<{
  logEntry: DockerEventLogEntry;
  showContainerName?: boolean;
}>();

const { containers } = useLoggingContext();

const nextContainer = computed(
  () =>
    [
      ...containers.value.filter(
        (c) =>
          c.host === containers.value[0].host &&
          c.created > logEntry.date &&
          c.name === containers.value[0].name &&
          c.state === "running",
      ),
    ].sort((a, b) => +a.created - +b.created)[0],
);

function redirectNow() {
  showToast(
    {
      title: t("alert.redirected.title"),
      message: t("alert.redirected.message", { containerId: nextContainer.value.id }),
      type: "info",
    },
    { expire: 5000 },
  );
  router.push({ name: "container-id", params: { id: nextContainer.value.id } });
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
