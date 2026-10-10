<template>
  <div class="flex min-h-full flex-col">
    <div class="space-y-6 p-4 pb-8">
      <div class="pr-20">
        <h2 class="truncate text-2xl font-bold">{{ container.name }}</h2>
        <p class="text-base-content/60 truncate font-mono text-sm">{{ container.image }}</p>
      </div>

      <!-- The facts people open inspect for: is it up, and if not, why did it go down. -->
      <div class="flex flex-wrap items-center gap-2">
        <span class="status-pill" :class="stateClass">{{ container.state }}</span>
        <span v-if="container.health" class="status-pill" :class="healthClass">{{ container.health }}</span>
        <template v-if="details">
          <span v-if="details.oomKilled" class="status-pill status-pill-error">{{ $t("inspect.oom-killed") }}</span>
          <span v-if="container.state !== 'running' && details.exitCode !== 0" class="status-pill status-pill-error">
            {{ $t("inspect.exit-code", { code: details.exitCode }) }}
          </span>
          <span v-if="details.restartCount > 0" class="status-pill status-pill-warning">
            {{ $t("inspect.restarts", { count: details.restartCount }) }}
          </span>
        </template>
      </div>

      <InlineNotice v-if="error" type="error">{{ $t("inspect.load-failed") }}</InlineNotice>

      <section class="space-y-3">
        <h3 class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">{{ $t("inspect.overview") }}</h3>
        <dl class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
          <div v-for="row in overview" :key="row.label" class="flex items-baseline gap-4 px-4 py-3">
            <dt class="text-base-content/60 w-32 shrink-0 text-sm">{{ row.label }}</dt>
            <dd class="min-w-0 flex-1 text-right font-mono text-xs wrap-anywhere">{{ row.value }}</dd>
          </div>
        </dl>
      </section>

      <section v-if="container.ports.length" class="space-y-3">
        <h3 class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">{{ $t("inspect.ports") }}</h3>
        <ul class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
          <li v-for="port in container.ports" :key="port" class="px-4 py-3 font-mono text-xs">
            {{ formatPort(port) }}
          </li>
        </ul>
      </section>

      <section v-if="container.mounts.length" class="space-y-3">
        <h3 class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">{{ $t("inspect.mounts") }}</h3>
        <ul class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
          <li
            v-for="mount in container.mounts"
            :key="mount.destination"
            class="flex items-center gap-3 px-4 py-3 font-mono text-xs"
          >
            <span class="min-w-0 flex-1 wrap-anywhere">
              <span class="text-base-content/60">{{ mount.source || mount.type }}</span>
              <span class="text-base-content/40"> → </span>
              <span class="font-semibold">{{ mount.destination }}</span>
            </span>
            <span class="status-pill shrink-0" :class="mount.rw ? 'status-pill-neutral' : 'status-pill-secondary'">
              {{ mount.rw ? "rw" : "ro" }}
            </span>
          </li>
        </ul>
      </section>

      <section v-if="details?.env.length" class="space-y-3">
        <h3 class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">{{ $t("inspect.env") }}</h3>
        <ul class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
          <li v-for="(entry, i) in details.env" :key="i" class="flex items-center gap-3 px-4 py-2 font-mono text-xs">
            <span class="shrink-0 font-semibold">{{ entry.key }}</span>
            <span class="text-base-content/60 min-w-0 flex-1 text-right wrap-anywhere">
              <template v-if="!details.envRevealed">••••••</template>
              <template v-else-if="revealed.has(i)">{{ entry.value }}</template>
              <template v-else-if="entry.value">••••••</template>
            </span>
            <button
              v-if="details.envRevealed && entry.value"
              type="button"
              class="icon-btn btn btn-ghost btn-xs btn-square"
              :title="revealed.has(i) ? $t('inspect.hide-value') : $t('inspect.show-value')"
              @click="toggle(i)"
            >
              <mdi:eye-off-outline v-if="revealed.has(i)" class="size-4 opacity-60" />
              <mdi:eye-outline v-else class="size-4 opacity-60" />
            </button>
          </li>
        </ul>
        <p v-if="!details.envRevealed" class="text-base-content/40 text-xs">{{ $t("inspect.env-hidden") }}</p>
      </section>

      <CollapsibleSection
        v-if="labels.length"
        v-model="labelsCollapsed"
        :title="$t('inspect.labels')"
        :count="labels.length"
      >
        <ul class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
          <li v-for="[key, value] in labels" :key="key" class="px-4 py-2 font-mono text-xs wrap-anywhere">
            <span class="text-base-content/60">{{ key }}</span>
            <span class="text-base-content/40">=</span>
            <span>{{ value }}</span>
          </li>
        </ul>
      </CollapsibleSection>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";

interface InspectDetails {
  restartPolicy?: string;
  restartCount: number;
  oomKilled: boolean;
  exitCode: number;
  networkMode?: string;
  env: { key: string; value?: string }[];
  envRevealed: boolean;
}

const { container } = defineProps<{ container: Container }>();
const { t } = useI18n();

const details = ref<InspectDetails>();
const error = ref(false);
const revealed = ref(new Set<number>());
const labelsCollapsed = ref(true);

watch(
  () => `${container.host}/${container.id}`,
  async () => {
    details.value = undefined;
    error.value = false;
    revealed.value = new Set();
    try {
      const response = await fetch(withBase(`/api/hosts/${container.host}/containers/${container.id}/inspect`));
      if (!response.ok) throw new Error(await response.text());
      details.value = await response.json();
    } catch (e) {
      console.error(e);
      error.value = true;
    }
  },
  { immediate: true },
);

function toggle(i: number) {
  const next = new Set(revealed.value);
  if (!next.delete(i)) next.add(i);
  revealed.value = next;
}

const stateClass = computed(() => {
  switch (container.state) {
    case "running":
      return "status-pill-success";
    case "paused":
    case "restarting":
      return "status-pill-warning";
    case "exited":
    case "dead":
      return "status-pill-error";
    default:
      return "status-pill-neutral";
  }
});

const healthClass = computed(() =>
  container.health === "healthy"
    ? "status-pill-success"
    : container.health === "unhealthy"
      ? "status-pill-error"
      : "status-pill-neutral",
);

const dateFormat = new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "medium" });

const overview = computed(() => {
  const rows = [
    { label: t("inspect.id"), value: container.id.slice(0, 12) },
    { label: t("inspect.command"), value: container.command },
    { label: t("inspect.created"), value: dateFormat.format(container.created) },
  ];
  if (container.state === "running") {
    rows.push({ label: t("inspect.started"), value: dateFormat.format(container.startedAt) });
  } else if (container.finishedAt.getTime() > 0) {
    rows.push({ label: t("inspect.finished"), value: dateFormat.format(container.finishedAt) });
  }
  if (details.value?.restartPolicy)
    rows.push({ label: t("inspect.restart-policy"), value: details.value.restartPolicy });
  if (details.value?.networkMode) rows.push({ label: t("inspect.network"), value: details.value.networkMode });
  rows.push({
    label: t("inspect.memory-limit"),
    value: container.memoryLimit > 0 ? formatBytes(container.memoryLimit) : t("inspect.unlimited"),
  });
  rows.push({
    label: t("inspect.cpu-limit"),
    value: container.cpuLimit > 0 ? `${container.cpuLimit}` : t("inspect.unlimited"),
  });
  return rows.filter((row) => row.value);
});

// Docker's `ip:host->container/proto`. A binding on every interface has no ip, so it
// reads as just the two ports.
const formatPort = (port: string) => port.replace(/^:/, "").replace("->", " → ");

const labels = computed(() => Object.entries(container.labels).sort(([a], [b]) => a.localeCompare(b)));
</script>
