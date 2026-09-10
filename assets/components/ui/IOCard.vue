<template>
  <div
    class="grid grid-cols-[auto_auto_1fr_auto_1fr] items-center gap-x-1.5 gap-y-1.5 px-3 py-1.5 text-[11.5px] leading-none tabular-nums max-md:hidden @max-5xl:hidden"
    :title="tooltip"
  >
    <template v-for="row in rows" :key="row.label">
      <span
        class="text-base-content/40 text-[10px] font-medium tracking-wider uppercase"
        :class="{ 'opacity-40': row.idle }"
        >{{ row.label }}</span
      >
      <PhArrowUp class="text-base-content/35 size-2.5" :class="{ 'opacity-40': row.idle }" />
      <span class="text-right" :class="{ 'text-base-content/40': !row.up }">{{ rate(row.up) }}</span>
      <PhArrowDown class="text-base-content/35 size-2.5" :class="{ 'opacity-40': row.idle }" />
      <span class="text-right" :class="{ 'text-base-content/40': !row.down }">{{ rate(row.down) }}</span>
    </template>
  </div>
</template>

<script lang="ts" setup>
import PhArrowUp from "~icons/ph/arrow-up";
import PhArrowDown from "~icons/ph/arrow-down";

const { networkRx, networkTx, diskRead, diskWrite } = defineProps<{
  networkRx: number;
  networkTx: number;
  diskRead: number;
  diskWrite: number;
}>();

const { t } = useI18n();

// Disk sits at zero for most containers, so an idle row is dimmed rather than
// removed: dropping it would resize the whole toolbar the moment a write lands.
const rows = computed(() => [
  { label: "NET", up: networkTx, down: networkRx, idle: !networkTx && !networkRx },
  { label: "DISK", up: diskWrite, down: diskRead, idle: !diskWrite && !diskRead },
]);

const rate = (bytes: number) => formatBytes(bytes, { short: true, decimals: 1 }) + "/s";

const tooltip = computed(
  () =>
    t("tooltip.network-io", { tx: formatBytes(networkTx), rx: formatBytes(networkRx) }) +
    "\n" +
    t("tooltip.disk-io", { write: formatBytes(diskWrite), read: formatBytes(diskRead) }),
);
</script>
