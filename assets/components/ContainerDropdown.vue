<template>
  <div class="dropdown">
    <button tabindex="0" role="button" class="btn btn-xs md:btn-sm"><slot /> <carbon:caret-down /></button>
    <ul tabindex="0" class="dropdown-content menu rounded-box bg-base-100 border-base-content/20 border shadow-sm">
      <li v-for="other in containers">
        <router-link :to="{ name: '/container/[id]', params: { id: other.id } }" class="text-nowrap">
          <div
            class="status data-[state=exited]:status-error data-[state=running]:status-success"
            :data-state="other.state"
          ></div>
          {{ other.name }}
          <div v-if="other.state === 'running'">running</div>
          <RelativeTime :date="other.finishedAt" class="text-base-content/70 text-xs" v-else />
        </router-link>
      </li>
    </ul>
  </div>
</template>
<script lang="ts" setup>
import { type Container } from "@/models/Container";
const { containers } = defineProps<{
  containers: Container[];
}>();
</script>
