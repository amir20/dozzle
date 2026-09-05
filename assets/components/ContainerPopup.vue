<template>
  <div class="flex w-64 flex-col gap-3">
    <div class="flex items-start gap-2">
      <ContainerIcon :state="container.state" :health="container.health" :slug="container.icon" class="mt-0.5 size-5" />
      <div class="min-w-0 flex-1">
        <div class="truncate font-semibold" :title="container.name">{{ container.name }}</div>
        <div class="text-base-content/50 truncate font-mono text-[11px]" :title="imageTag">{{ imageTag }}</div>
      </div>
      <ContainerHealth v-if="container.health" :health="container.health" />
    </div>

    <div v-if="isRunning" class="grid grid-cols-[auto_1fr] items-center gap-x-2 gap-y-1">
      <span class="text-base-content/50 text-[11px] uppercase">{{ $t("label.cpu") }}</span>
      <ContainerStatCell :container="container" type="cpu" :host="host" />
      <span class="text-base-content/50 text-[11px] uppercase">{{ $t("label.mem") }}</span>
      <ContainerStatCell :container="container" type="mem" :host="host" />
    </div>

    <div class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
      <template v-for="row in rows" :key="row.label">
        <span class="text-base-content/50 uppercase">{{ row.label }}</span>
        <span class="truncate text-right font-medium" :title="row.title ?? row.value">{{ row.value }}</span>
      </template>
      <template v-if="isRunning">
        <span class="text-base-content/50 uppercase">{{ $t("popup.started") }}</span>
        <RelativeTime :date="container.startedAt" class="truncate text-right font-medium" />
      </template>
      <template v-else-if="container.finishedAt.getFullYear() > 0">
        <span class="text-base-content/50 uppercase">{{ $t("popup.finished") }}</span>
        <RelativeTime :date="container.finishedAt" class="truncate text-right font-medium" />
      </template>
    </div>

    <ContainerLink
      v-if="container.url"
      :container="container"
      class="border-base-content/10 text-primary! border-t pt-2 text-xs hover:underline"
    >
      <span class="flex-1 truncate">{{ shortUrl }}</span>
    </ContainerLink>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";

const { container } = defineProps<{
  container: Container;
}>();

const { t } = useI18n();
const { hosts } = useHosts();

const host = computed(() => hosts.value[container.host]);
const isRunning = computed(() => container.state === "running");
const imageTag = computed(() => container.image.replace(/@sha.*/, ""));
const shortUrl = computed(() => container.url?.replace(/^https?:\/\//, "").replace(/\/$/, ""));

const rows = computed(() => {
  const rows: { label: string; value: string; title?: string }[] = [
    { label: t("label.status"), value: container.health ?? container.state },
  ];

  if (config.hosts.length > 1) {
    rows.push({ label: t("label.host"), value: container.hostLabel });
  }

  if (container.customGroup) {
    rows.push({ label: t("popup.group"), value: container.customGroup });
  }

  const ports = container.portMappings;
  if (ports.length > 0) {
    const mapped = ports.map(({ host, container }) => `${host}→${container}`);
    rows.push({
      label: t("popup.ports"),
      value: mapped.length > 2 ? `${mapped.slice(0, 2).join(", ")} +${mapped.length - 2}` : mapped.join(", "),
      title: mapped.join(", "),
    });
  }

  return rows;
});
</script>
