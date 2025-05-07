<template>
  <table class="w-full border-separate border-spacing-x-1">
    <tbody>
      <tr>
        <th class="text-right font-light capitalize">STATE</th>
        <td class="font-semibold uppercase">{{ container.state }}</td>
      </tr>
      <tr v-if="container.startedAt.getFullYear() > 0">
        <th class="text-right font-light capitalize">STARTED</th>
        <td class="font-semibold">
          <RelativeTime :date="container.startedAt" />
        </td>
      </tr>
      <tr v-if="container.state != 'running' && container.finishedAt.getFullYear() > 0">
        <th class="text-right font-light capitalize">FINISHED</th>
        <td class="font-semibold">
          <RelativeTime :date="container.finishedAt" />
        </td>
      </tr>
      <tr v-if="container.state == 'running'">
        <th class="text-right font-light capitalize">Load</th>
        <td class="font-semibold">{{ container.stat.cpu.toFixed(2) }}%</td>
      </tr>
      <tr v-if="container.state == 'running'">
        <th class="text-right font-light capitalize">MEM</th>
        <td class="font-semibold">{{ formatBytes(container.stat.memoryUsage) }}</td>
      </tr>
    </tbody>
  </table>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";

const { container } = defineProps<{
  container: Container;
}>();
</script>
