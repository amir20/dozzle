<template>
  <Popover
    hover
    placement="bottom-end"
    panel-class="rounded-box border-base-content/10 bg-base-200 w-72 border p-3 text-xs shadow-lg"
  >
    <template #trigger>
      <!-- empty title: the host chip's own "Host" tooltip would pop up over the panel -->
      <button type="button" title="" class="hover:bg-base-content/10 -mx-1 rounded px-1 transition-colors">
        {{ $t("label.reclaimable") }}
        <span class="text-base-content/80 font-mono">{{ size(total) }}</span>
      </button>
    </template>

    <div class="flex items-baseline justify-between gap-2">
      <span class="text-base-content/60 font-semibold tracking-wide uppercase">{{ $t("label.reclaimable") }}</span>
      <span class="font-mono text-sm font-semibold">{{ size(total) }}</span>
    </div>

    <ul class="mt-3 space-y-2.5">
      <li v-for="row in rows" :key="row.key" :class="{ 'opacity-50': row.size === 0 }">
        <div class="flex items-center gap-2">
          <component :is="row.icon" class="size-4 shrink-0 opacity-60" />
          <span class="min-w-0 flex-1 truncate">{{ row.label }}</span>
          <span v-if="row.count !== undefined" class="text-base-content/40 font-mono">{{ row.count }}</span>
          <span class="w-16 text-right font-mono">{{ size(row.size) }}</span>
        </div>
        <!-- each kind's share of the total, neutral like the host card's disk bar -->
        <div class="bg-base-content/10 mt-1 ml-6 h-1 overflow-hidden rounded-full">
          <div
            class="bg-base-content/40 h-full rounded-full transition-[width] duration-500 motion-reduce:transition-none"
            :style="{ width: share(row.size) }"
          ></div>
        </div>
      </li>
    </ul>

    <div class="bg-base-content/10 my-3 h-px"></div>
    <i18n-t keypath="tooltip.reclaimable-source" tag="p" class="text-base-content/40">
      <template #command>
        <code class="font-mono">docker system df</code>
      </template>
    </i18n-t>
  </Popover>
</template>

<script lang="ts" setup>
import type { Component } from "vue";
import type { Reclaimable } from "@/stores/hosts";
import PhStack from "~icons/ph/stack";
import PhDatabase from "~icons/ph/database";
import PhStopCircle from "~icons/ph/stop-circle";
import PhHammer from "~icons/ph/hammer";

const { reclaimable } = defineProps<{ reclaimable: Reclaimable }>();

const { t } = useI18n();

type Row = { key: string; icon: Component; label: string; count?: number; size: number };

// A fixed order, so the same kind sits in the same place on every host.
const rows = computed<Row[]>(() => [
  {
    key: "images",
    icon: PhStack,
    label: t("label.unused-images"),
    count: reclaimable.images,
    size: reclaimable.imagesSize,
  },
  {
    key: "volumes",
    icon: PhDatabase,
    label: t("label.unused-volumes"),
    count: reclaimable.volumes,
    size: reclaimable.volumesSize,
  },
  {
    key: "containers",
    icon: PhStopCircle,
    label: t("label.stopped-containers"),
    count: reclaimable.containers,
    size: reclaimable.containersSize,
  },
  { key: "build-cache", icon: PhHammer, label: t("label.build-cache"), size: reclaimable.buildCacheSize },
]);

const total = computed(() => rows.value.reduce((sum, row) => sum + row.size, 0));

const size = (bytes: number) => formatBytes(bytes, { decimals: 1 });
const share = (bytes: number) => (total.value ? `${(bytes / total.value) * 100}%` : "0%");
</script>
