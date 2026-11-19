<template>
  <Popover sheet panel-class="rounded-box bg-base-100 border-base-content/20 border p-1 shadow-sm">
    <template #trigger>
      <button type="button" class="btn btn-xs md:btn-sm"><slot /> <carbon:caret-down /></button>
    </template>
    <ul class="menu w-full p-0">
      <li v-for="other in sorted">
        <router-link :to="{ name: '/container/[id]', params: { id: other.id } }" class="text-nowrap">
          <div
            class="status data-[state=exited]:status-error data-[state=running]:status-success data-[state=paused]:status-warning"
            :data-state="other.state"
          ></div>
          {{ other.name }}
          <span v-if="collides.has(other.name)" class="text-base-content/50 text-xs">{{ hostName(other) }}</span>
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

// Every task of a global swarm service has the same name, one per node, so the
// host is shown for a name that runs on more than one host (#5681).
const collides = computed(() => {
  const hosts = new Map<string, Set<string>>();
  for (const c of containers) hosts.set(c.name, (hosts.get(c.name) ?? new Set()).add(c.host));
  return new Set([...hosts].filter(([, h]) => h.size > 1).map(([name]) => name));
});

// The host's name, or its id until the host list has loaded.
const hostName = (c: Container) => c.hostLabel ?? c.host;

// Same-name entries sit together, in host order.
const sorted = computed(() =>
  [...containers].sort((a, b) => a.name.localeCompare(b.name) || hostName(a).localeCompare(hostName(b))),
);
</script>
