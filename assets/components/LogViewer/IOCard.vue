<template>
  <div
    class="bg-base-content/[0.06] grid min-w-0 grid-cols-[auto_auto_3.5rem_auto_3.5rem] items-center gap-x-1.5 gap-y-1 rounded-md px-2.5 py-1.5 text-[12.5px] leading-none tabular-nums max-md:hidden @max-5xl:hidden"
    :title="tooltip"
  >
    <PhNetwork class="text-base-content/60 size-3.5" />
    <PhArrowUp class="text-primary text-[10px]" />
    <span class="text-right">{{ formatBytes(networkTx, { short: true, decimals: 1 }) }}/s</span>
    <PhArrowDown class="text-secondary text-[10px]" />
    <span class="text-right">{{ formatBytes(networkRx, { short: true, decimals: 1 }) }}/s</span>

    <PhHardDrives class="text-base-content/60 size-3.5" />
    <PhArrowUp class="text-primary text-[10px]" />
    <span class="text-right">{{ formatBytes(diskWrite, { short: true, decimals: 1 }) }}/s</span>
    <PhArrowDown class="text-secondary text-[10px]" />
    <span class="text-right">{{ formatBytes(diskRead, { short: true, decimals: 1 }) }}/s</span>
  </div>
</template>

<script lang="ts" setup>
// @ts-ignore
import PhNetwork from "~icons/ph/network";
// @ts-ignore
import PhHardDrives from "~icons/ph/hard-drives";
// @ts-ignore
import PhArrowUp from "~icons/ph/arrow-up";
// @ts-ignore
import PhArrowDown from "~icons/ph/arrow-down";

const { networkRx, networkTx, diskRead, diskWrite } = defineProps<{
  networkRx: number;
  networkTx: number;
  diskRead: number;
  diskWrite: number;
}>();

const { t } = useI18n();
const tooltip = computed(
  () =>
    t("tooltip.network-io", { tx: formatBytes(networkTx), rx: formatBytes(networkRx) }) +
    "\n" +
    t("tooltip.disk-io", { write: formatBytes(diskWrite), read: formatBytes(diskRead) }),
);
</script>
