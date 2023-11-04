<template>
  <div class="flex flex-col gap-16 px-4 pt-8 md:px-8">
    <section>
      <div class="flex items-center justify-end gap-4">
        <template v-if="config.pages">
          <router-link
            :to="{ name: 'content-id', params: { id: page.id } }"
            :title="page.title"
            v-for="page in config.pages"
            :key="page.id"
            class="link-primary"
          >
            {{ page.title }}
          </router-link>
        </template>
        <template v-if="config.user">
          <div v-if="config.authProvider === 'simple'">
            <button @click.prevent="logout()" class="link-primary">{{ $t("button.logout") }}</button>
          </div>
          <div>
            {{ config.user.name ? config.user.name : config.user.email }}
          </div>
          <img class="h-10 w-10 rounded-full p-1 ring-2 ring-base-content/50" :src="config.user.avatar" />
        </template>
      </div>
    </section>
    <section>
      <div class="stats grid bg-base-lighter shadow">
        <div class="stat">
          <div class="stat-value">{{ runningContainers.length }} / {{ containers.length }}</div>
          <div class="stat-title">{{ $t("label.running") }} / {{ $t("label.total-containers") }}</div>
        </div>
        <div class="stat">
          <div class="stat-value">{{ totalCpu }}%</div>
          <div class="stat-title">{{ $t("label.total-cpu-usage") }}</div>
        </div>
        <div class="stat">
          <div class="stat-value">{{ formatBytes(totalMem) }}</div>
          <div class="stat-title">{{ $t("label.total-mem-usage") }}</div>
        </div>

        <div class="stat">
          <div class="stat-value">{{ version }}</div>
          <div class="stat-title">{{ $t("label.dozzle-version") }}</div>
        </div>
      </div>
    </section>

    <section>
      <container-table :containers="runningContainers"></container-table>
    </section>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";

const { t } = useI18n();
const { version } = config;
const containerStore = useContainerStore();
const { containers, ready } = storeToRefs(containerStore) as unknown as {
  containers: Ref<Container[]>;
  ready: Ref<boolean>;
};

const mostRecentContainers = $computed(() => containers.value.toSorted((a, b) => +b.created - +a.created));
const runningContainers = $computed(() => mostRecentContainers.filter((c) => c.state === "running"));

let totalCpu = $ref(0);
useIntervalFn(
  () => {
    totalCpu = runningContainers.reduce((acc, c) => acc + c.stat.cpu, 0);
  },
  1000,
  { immediate: true },
);

let totalMem = $ref(0);
useIntervalFn(
  () => {
    totalMem = runningContainers.reduce((acc, c) => acc + c.stat.memoryUsage, 0);
  },
  1000,
  { immediate: true },
);

watchEffect(() => {
  if (ready.value) {
    setTitle(t("title.dashboard", { count: runningContainers.length }));
  }
});

async function logout() {
  await fetch(withBase("/api/token"), {
    method: "DELETE",
  });

  location.reload();
}
</script>
<style lang="postcss" scoped>
:deep(tr td) {
  padding-top: 1em;
  padding-bottom: 1em;
}

.stat > div {
  @apply text-center;
}

.stat-value {
  @apply font-light;
}

.stat-title {
  @apply font-light;
}

.section + .section {
  padding-top: 0;
}
</style>
