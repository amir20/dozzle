<template>
  <li>
    <button
      type="button"
      class="hover:bg-base-300/50 flex w-full items-center gap-3 px-4 py-3 text-left transition-colors"
      :aria-expanded="open"
      @click="open = !open"
    >
      <mdi:chevron-down class="size-4 shrink-0 opacity-60 transition-transform" :class="{ '-rotate-90': !open }" />
      <span class="min-w-0 flex-1 truncate font-mono text-xs">
        <span class="font-semibold">{{ network.name }}</span>
        <span v-if="details?.driver" class="text-base-content/40"> · {{ details.driver }}</span>
      </span>
      <span v-if="others > 0" class="status-pill status-pill-neutral shrink-0">
        {{ $t("inspect.container-count", others + 1) }}
      </span>
      <span v-if="network.ip" class="text-base-content/60 shrink-0 font-mono text-xs">{{ network.ip }}</span>
    </button>

    <div v-if="open" class="space-y-4 px-4 pb-4 pl-11">
      <p v-if="error" class="text-base-content/60 text-xs">{{ $t("inspect.details-failed") }}</p>
      <template v-else-if="details">
        <div v-for="subnet in details.subnets" :key="subnet.subnet" class="space-y-3">
          <dl class="grid grid-cols-2 gap-x-6 gap-y-1">
            <div class="min-w-0">
              <dt class="text-base-content/60 text-xs">{{ $t("inspect.subnet") }}</dt>
              <dd class="mt-1 truncate font-mono text-sm">{{ subnet.subnet }}</dd>
            </div>
            <div v-if="subnet.gateway" class="min-w-0">
              <dt class="text-base-content/60 text-xs">{{ $t("inspect.gateway") }}</dt>
              <dd class="mt-1 truncate font-mono text-sm">{{ subnet.gateway }}</dd>
            </div>
          </dl>
          <!-- Docker only reports pool usage from Engine 28; older daemons send zeros. -->
          <UsageMeter
            v-if="subnet.inUse + subnet.available > 0"
            :label="$t('inspect.ip-usage')"
            :used="subnet.inUse"
            :limit="subnet.inUse + subnet.available"
            compact
            hide-percent
          />
        </div>
        <div class="space-y-1">
          <h4 class="text-base-content/60 text-xs">{{ $t("inspect.connected-containers") }}</h4>
          <InspectContainerList :containers="details.containers" :current-id="container.id" class="-mx-2 text-sm" />
        </div>
      </template>
      <div v-else class="flex justify-center py-2">
        <span class="loading loading-spinner loading-sm opacity-60"></span>
      </div>
    </div>
  </li>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";

interface NetworkDetails {
  driver: string;
  subnets: { subnet: string; gateway?: string; inUse: number; available: number }[];
  containers: { id: string; name: string; ip?: string }[];
}

const { container, network } = defineProps<{ container: Container; network: { name: string; ip?: string } }>();

const open = ref(false);
const details = ref<NetworkDetails>();
const error = ref(false);

// Loaded with the drawer rather than on expand: a container sits on one or two
// networks, and the driver and neighbour count belong on the collapsed row.
async function load() {
  try {
    const response = await fetch(
      withBase(`/api/hosts/${container.host}/containers/${container.id}/networks/${encodeURIComponent(network.name)}`),
    );
    if (!response.ok) throw new Error(await response.text());
    details.value = await response.json();
  } catch (e) {
    console.error(e);
    error.value = true;
  }
}
load();

const others = computed(() => details.value?.containers.filter((c) => c.id !== container.id).length ?? 0);
</script>
