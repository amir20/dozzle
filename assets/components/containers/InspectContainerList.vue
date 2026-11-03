<template>
  <ul class="space-y-0.5">
    <li v-for="entry in sorted" :key="entry.id">
      <!-- Another container's inspect opens in this same drawer, so a network or volume
           can be followed from one container to the next without losing the drawer. -->
      <button
        v-if="entry.id !== currentId && byId[entry.id]"
        type="button"
        class="hover:bg-base-300 flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left transition-colors"
        @click="showDrawer(ContainerInspect, { container: byId[entry.id] })"
      >
        <span class="size-1.5 shrink-0 rounded-full" :class="dotClass(entry.id)"></span>
        <span class="min-w-0 flex-1 truncate">{{ entry.name }}</span>
        <span v-if="entry.ip" class="text-base-content/60 shrink-0 font-mono text-xs">{{ entry.ip }}</span>
        <mdi:chevron-right class="size-3.5 shrink-0 opacity-40" />
      </button>
      <div v-else class="flex items-center gap-2 px-2 py-1.5">
        <span class="size-1.5 shrink-0 rounded-full" :class="dotClass(entry.id)"></span>
        <span class="min-w-0 flex-1 truncate">
          {{ entry.name }}
          <span v-if="entry.id === currentId" class="text-base-content/40 text-xs">
            · {{ $t("inspect.this-container") }}
          </span>
        </span>
        <span v-if="entry.ip" class="text-base-content/60 shrink-0 font-mono text-xs">{{ entry.ip }}</span>
        <span class="size-3.5 shrink-0"></span>
      </div>
    </li>
  </ul>
</template>

<script lang="ts" setup>
import ContainerInspect from "./ContainerInspect.vue";

const { containers, currentId } = defineProps<{
  containers: { id: string; name: string; ip?: string }[];
  currentId: string;
}>();

const showDrawer = useDrawer();
const { allContainersById: byId } = storeToRefs(useContainerStore());

// This container first, then the rest by name, so the list reads as "me and my neighbours".
const sorted = computed(() =>
  [...containers].sort((a, b) => (a.id === currentId ? -1 : b.id === currentId ? 1 : a.name.localeCompare(b.name))),
);

const dotClass = (id: string) => (byId.value[id]?.state === "running" ? "bg-success" : "bg-base-content/30");
</script>
