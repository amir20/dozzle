<template>
  <!-- Hover opens it only while nothing is in progress: once a box is ticked or a
       delete is being confirmed, drifting off the panel must not throw that away. -->
  <Popover
    :hover="step === 'view' && !touched"
    :close-on-select="false"
    placement="bottom-end"
    panel-class="rounded-box border-base-content/10 bg-base-200 w-72 border p-3 text-xs shadow-lg"
    @close="onClose"
  >
    <template #trigger>
      <button type="button" class="hover:bg-base-content/10 -mx-1 rounded px-1 transition-colors">
        {{ $t("label.reclaimable") }}
        <span class="text-base-content/80 font-mono [word-spacing:-0.4ch]">{{ size(total) }}</span>
      </button>
    </template>

    <template v-if="step === 'confirm'">
      <div class="flex items-start gap-3">
        <div class="bg-error/10 text-error flex size-8 shrink-0 items-center justify-center rounded-full">
          <mdi:trash-can-outline class="size-4" />
        </div>
        <div class="min-w-0">
          <div class="text-sm font-semibold">
            {{ $t("reclaim.confirm-title", { size: size(selectedTotal), host: host.name }) }}
          </div>
          <div class="text-base-content/60 mt-0.5">{{ $t("reclaim.cannot-undo") }}</div>
        </div>
      </div>

      <ul class="border-base-content/15 bg-base-200/40 divide-base-content/10 mt-3 divide-y rounded-lg border">
        <li v-for="row in selectedRows" :key="row.key" class="flex items-center gap-2 px-2 py-1.5">
          <component :is="row.icon" class="size-4 shrink-0 opacity-60" />
          <span class="min-w-0 flex-1 truncate">{{ row.label }}</span>
          <span v-if="row.count !== undefined" class="text-base-content/40 font-mono">{{ row.count }}</span>
          <span class="w-16 text-right font-mono">{{ size(row.size) }}</span>
        </li>
      </ul>

      <p v-if="picked.images" class="text-base-content/40 mt-2">{{ $t("reclaim.images-repull") }}</p>

      <div v-if="picked.volumes" class="border-base-content/15 mt-3 flex items-start gap-2 rounded-lg border p-2">
        <div class="bg-warning/10 text-warning flex size-6 shrink-0 items-center justify-center rounded-full">
          <mdi:alert-outline class="size-3.5" />
        </div>
        <div class="min-w-0">
          <p>{{ $t("reclaim.volumes-warning", { count: reclaimable.volumes }, reclaimable.volumes) }}</p>
          <label class="mt-2 flex items-center gap-2">
            <input v-model="volumesConfirmed" type="checkbox" class="checkbox checkbox-error checkbox-sm" />
            {{ $t("reclaim.volumes-confirm") }}
          </label>
        </div>
      </div>

      <div class="mt-3 flex justify-end gap-2">
        <button type="button" class="btn btn-sm" @click="step = 'view'">{{ $t("button.cancel") }}</button>
        <button
          type="button"
          class="btn btn-error btn-sm"
          :disabled="picked.volumes && !volumesConfirmed"
          @click="prune"
        >
          {{ $t("reclaim.delete") }}
        </button>
      </div>
    </template>

    <template v-else-if="step === 'running'">
      <div class="flex items-center gap-3 py-1">
        <span class="loading loading-spinner loading-sm text-base-content/60"></span>
        <span class="text-sm">{{ $t("reclaim.deleting", { size: size(selectedTotal) }) }}</span>
      </div>
    </template>

    <template v-else>
      <div v-if="step === 'done'" class="mb-3 flex items-center gap-3">
        <div class="bg-success/10 text-success flex size-8 shrink-0 items-center justify-center rounded-full">
          <mdi:check class="size-4" />
        </div>
        <div class="text-sm font-semibold">
          {{ $t("reclaim.freed", { size: size(freed) }) }}
        </div>
      </div>
      <div v-else-if="step === 'error'" class="mb-3 flex items-start gap-3">
        <div class="bg-error/10 text-error flex size-8 shrink-0 items-center justify-center rounded-full">
          <mdi:alert-circle-outline class="size-4" />
        </div>
        <div class="min-w-0">
          <div class="text-sm font-semibold">{{ $t("reclaim.failed") }}</div>
          <div class="text-base-content/60 mt-0.5 break-words">{{ error }}</div>
        </div>
      </div>

      <div class="flex items-baseline justify-between gap-2">
        <span class="text-base-content/60 font-semibold tracking-wide uppercase">{{ $t("label.reclaimable") }}</span>
        <span class="font-mono text-sm font-semibold">{{ size(total) }}</span>
      </div>

      <ul class="mt-3 space-y-2.5">
        <li v-for="row in rows" :key="row.key" :class="{ 'opacity-50': row.empty }">
          <component :is="canPrune ? 'label' : 'div'" class="flex items-center gap-2">
            <input
              v-if="canPrune"
              type="checkbox"
              class="checkbox checkbox-primary checkbox-sm"
              :checked="picked[row.key] && !row.empty"
              :disabled="row.empty"
              @change="pick(row.key, ($event.target as HTMLInputElement).checked)"
            />
            <component :is="row.icon" v-else class="size-4 shrink-0 opacity-60" />
            <span class="min-w-0 flex-1 truncate">{{ row.label }}</span>
            <span v-if="row.count !== undefined" class="text-base-content/40 font-mono">{{ row.count }}</span>
            <span class="w-16 text-right font-mono">{{ size(row.size) }}</span>
          </component>
          <!-- each kind's share of the total, neutral like the host card's disk bar -->
          <div class="bg-base-content/10 mt-1 h-1 overflow-hidden rounded-full" :class="canPrune ? 'ml-7' : 'ml-6'">
            <div
              class="bg-base-content/40 h-full rounded-full transition-[width] duration-500 motion-reduce:transition-none"
              :style="{ width: share(row.size) }"
            ></div>
          </div>
        </li>
      </ul>

      <div class="bg-base-content/10 my-3 h-px"></div>
      <i18n-t scope="global" keypath="tooltip.reclaimable-source" tag="p" class="text-base-content/40">
        <template #command>
          <code class="font-mono">docker system df</code>
        </template>
      </i18n-t>
      <button
        v-if="canPrune"
        type="button"
        class="btn btn-primary btn-sm mt-3 w-full"
        :disabled="selectedRows.length === 0"
        @click="confirm"
      >
        <mdi:broom class="size-4" />
        {{ $t("reclaim.clean-up") }}
        <span v-if="selectedRows.length" class="font-mono">{{ size(selectedTotal) }}</span>
      </button>
      <p v-if="actionsOff" class="text-base-content/40 mt-2">
        {{ $t("reclaim.enable-actions") }}
        <a href="https://dozzle.dev/guide/actions" target="_blank" rel="noopener" class="link">
          {{ $t("hint.learn-more") }}
        </a>
      </p>
    </template>
  </Popover>
</template>

<script lang="ts" setup>
import type { Component } from "vue";
import type { Host, Reclaimable } from "@/stores/hosts";
import PhStack from "~icons/ph/stack";
import PhDatabase from "~icons/ph/database";
import PhStopCircle from "~icons/ph/stop-circle";
import PhHammer from "~icons/ph/hammer";

const { host } = defineProps<{ host: Host }>();

const { t } = useI18n();
const { updateHostMetrics } = useHosts();
const { enableActions } = config;

type Kind = "images" | "volumes" | "containers" | "buildCache";
type Row = { key: Kind; icon: Component; label: string; count?: number; size: number; empty: boolean };
type Step = "view" | "confirm" | "running" | "done" | "error";

const EMPTY: Reclaimable = {
  images: 0,
  imagesSize: 0,
  volumes: 0,
  volumesSize: 0,
  containers: 0,
  containersSize: 0,
  buildCacheSize: 0,
};
const reclaimable = computed(() => host.reclaimable ?? EMPTY);

// k8s never reports reclaimable space, but say so rather than lean on that.
const canPrune = computed(() => enableActions && host.type !== "k8s");
// k8s can't prune at all, so only a docker host points at the setting.
const actionsOff = computed(() => !enableActions && host.type !== "k8s");

const step = ref<Step>("view");
// Volumes start unticked: an unused named volume can be a database between redeploys.
const picked = reactive<Record<Kind, boolean>>({ images: true, volumes: false, containers: true, buildCache: true });
const touched = ref(false);
const volumesConfirmed = ref(false);
const freed = ref(0);
const error = ref("");

// A fixed order, so the same kind sits in the same place on every host.
const rows = computed<Row[]>(() => {
  const r = reclaimable.value;
  return [
    { key: "images", icon: PhStack, label: t("label.unused-images"), count: r.images, size: r.imagesSize },
    { key: "volumes", icon: PhDatabase, label: t("label.unused-volumes"), count: r.volumes, size: r.volumesSize },
    {
      key: "containers",
      icon: PhStopCircle,
      label: t("label.stopped-containers"),
      count: r.containers,
      size: r.containersSize,
    },
    { key: "buildCache", icon: PhHammer, label: t("label.build-cache"), size: r.buildCacheSize },
  ].map((row) => ({ ...row, empty: row.size === 0 && !row.count }) as Row);
});

const total = computed(() => rows.value.reduce((sum, row) => sum + row.size, 0));
const selectedRows = computed(() => rows.value.filter((row) => picked[row.key] && !row.empty));
const selectedTotal = computed(() => selectedRows.value.reduce((sum, row) => sum + row.size, 0));

const size = (bytes: number) => formatBytes(bytes, { decimals: 1 });
const share = (bytes: number) => (total.value ? `${(bytes / total.value) * 100}%` : "0%");

// An empty kind shows unticked but keeps its pick, so it is picked again once it fills.
function pick(kind: Kind, value: boolean) {
  picked[kind] = value;
  touched.value = true;
}

function confirm() {
  volumesConfirmed.value = false;
  step.value = "confirm";
}

async function prune() {
  const body = Object.fromEntries(selectedRows.value.map((row) => [row.key, true]));
  step.value = "running";
  try {
    const response = await fetch(withBase(`/api/hosts/${encodeURIComponent(host.id)}/prune`), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      // 501: an agent older than this Dozzle, which does not know the call
      error.value = response.status === 501 ? t("reclaim.agent-too-old") : (await response.text()).trim();
      step.value = "error";
      return;
    }
    const result = (await response.json()) as { spaceReclaimed: number; reclaimable: Reclaimable | null };
    freed.value = result.spaceReclaimed;
    if (result.reclaimable) updateHostMetrics({ id: host.id, reclaimable: result.reclaimable });
    step.value = "done";
  } catch {
    error.value = t("error.something-went-wrong");
    step.value = "error";
  }
}

// Closing walks away from a confirm or a result. A delete still running carries on,
// and its result is waiting when the panel opens again.
function onClose() {
  if (step.value === "running") return;
  step.value = "view";
  touched.value = false;
}
</script>
