<template>
  <LogItem :logEntry>
    <div class="whitespace-pre-wrap" :data-event="logEntry.event" v-html="logEntry.message"></div>
  </LogItem>

  <!--
    <div>
      <router-link
        :to="{ name: '/container/[id]', params: { id: nextContainer.id } }"
        class="btn btn-primary btn-sm"
        v-else
      >
        {{ $t("button.redirect") }}
      </router-link>
    -->
</template>
<script lang="ts" setup>
import { ContainerEventLogEntry } from "@/models/LogEntry";

const { logEntry } = defineProps<{
  logEntry: ContainerEventLogEntry;
  showContainerName?: boolean;
}>();

// function redirectNow() {
//   showToast(
//     {
//       title: t("alert.redirected.title"),
//       message: t("alert.redirected.message", { containerId: nextContainer.value?.id }),
//       type: "info",
//     },
//     { expire: 5000 },
//   );
//   router.push({ name: "/container/[id]", params: { id: nextContainer.value?.id } });
// }
</script>

<style scoped>
@import "@/main.css" reference;
[data-event="container-stopped"] {
  @apply text-red;
}
[data-event="container-started"] {
  @apply text-green;
}
</style>
