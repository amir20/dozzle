<template>
  <Popover panel-class="rounded-box bg-base-100 border-base-content/20 border p-1 shadow-sm">
    <template #trigger>
      <button type="button" class="btn btn-xs md:btn-sm"><slot /> <carbon:caret-down /></button>
    </template>
    <ul class="menu w-full p-0">
      <li v-for="other in containers">
        <router-link :to="{ name: '/container/[id]', params: { id: other.id } }" class="text-nowrap">
          <div
            class="status data-[state=exited]:status-error data-[state=running]:status-success data-[state=paused]:status-warning"
            :data-state="other.state"
          ></div>
          {{ other.name }}
          <div v-if="other.state === 'running'">running</div>
          <div v-else-if="other.state === 'paused'">paused</div>
          <RelativeTime :date="other.finishedAt" class="text-base-content/70 text-xs" v-else />
        </router-link>
      </li>
    </ul>
  </Popover>
</template>
<script lang="ts" setup>
import { type Container } from "@/models/Container";
const { containers } = defineProps<{
  containers: Container[];
}>();
</script>
