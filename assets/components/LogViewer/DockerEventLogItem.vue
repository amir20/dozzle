<template>
  <div class="flex-1 font-sans text-[0.9rem]">
    <span class="whitespace-pre-wrap" :data-event="logEntry.event" v-html="logEntry.message"></span>
    <div
      class="alert alert-info mt-8 w-auto text-[1rem] md:mx-auto md:w-1/2"
      v-if="nextContainer && logEntry.event === 'container-stopped'"
    >
      <carbon:information class="h-6 w-6 shrink-0 stroke-current" />
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
  </div>
</template>
<script lang="ts" setup>
import { DockerEventLogEntry } from "@/models/LogEntry";
const router = useRouter();
const { showToast } = useToast();
const { t } = useI18n();

const { logEntry } = defineProps<{
  logEntry: DockerEventLogEntry;
}>();

const store = useContainerStore();
const { containers } = storeToRefs(store);
const { container } = useContainerContext();

const nextContainer = computed(
  () =>
    containers.value
      .filter(
        (c) =>
          c.host === container.value.host &&
          c.created > logEntry.date &&
          c.name === container.value.name &&
          c.state === "running",
      )
      .toSorted((a, b) => +a.created - +b.created)[0],
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
