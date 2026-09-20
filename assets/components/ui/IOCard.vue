<template>
  <div
    class="grid grid-cols-[auto_auto_7ch_auto_7ch] items-center gap-1.5 px-3 py-1.5 text-[11.5px] leading-none tabular-nums max-md:hidden @max-5xl:hidden"
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
// The value columns are fixed rather than 1fr: an auto-sized column resized on
// almost every tick and walked the numbers (and the card, and its neighbours)
// left and right.
//
// Fixed means reserving the worst case, so `rate` caps the precision to keep that
// case short. Three significant figures puts the ceiling at 7 characters; the raw
// two-decimal form reached "1023.9K/s" and forced 9ch of mostly empty space for a
// value that is almost never on screen.
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

const rate = (bytes: number) => {
  // Below one byte the log would go negative and formatBytes would index past the
  // start of its unit table. Rates are whole byte deltas, so this is only a guard.
  if (bytes < 1) return "0B/s";
  // Drop the decimal once the scaled value reaches three digits: "105K/s" reads
  // the same as "105.3K/s" at this size and costs two fewer characters.
  const scaled = bytes / 1024 ** Math.floor(Math.log(bytes) / Math.log(1024));
  return formatBytes(bytes, { short: true, decimals: scaled >= 100 ? 0 : 1 }) + "/s";
};

const tooltip = computed(
  () =>
    t("tooltip.network-io", { tx: formatBytes(networkTx), rx: formatBytes(networkRx) }) +
    "\n" +
    t("tooltip.disk-io", { write: formatBytes(diskWrite), read: formatBytes(diskRead) }),
);
</script>
