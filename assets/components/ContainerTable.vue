<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-row">
      <div v-if="Object.keys(hosts).length > 1" class="flex-1">
        <div role="tablist" class="tabs-boxed tabs block" v-if="Object.keys(hosts).length < 4">
          <input
            type="radio"
            name="host"
            role="tab"
            class="tab rounded-sm!"
            aria-label="Show All"
            v-model="selectedHost"
            :value="null"
          />
          <input
            type="radio"
            name="host"
            role="tab"
            class="tab rounded-sm!"
            :aria-label="host.name"
            v-for="host in hosts"
            :value="host.id"
            :key="host.id"
            v-model="selectedHost"
          />
        </div>

        <DropdownMenu
          class="btn-sm"
          v-model="selectedHost"
          :options="[
            { label: 'Show All', value: null },
            ...Object.values(hosts).map((host) => ({ label: host.name, value: host.id })),
          ]"
          v-else
        />
      </div>
      <div class="flex flex-1 items-center justify-end gap-2">
        <div v-show="ready && containers.length > pageSizes[0]">
          {{ $t("label.per-page") }}

          <DropdownMenu
            class="dropdown-left btn-xs md:btn-sm"
            v-model="perPage"
            :options="pageSizes.map((i) => ({ label: i.toLocaleString(), value: i }))"
          />
        </div>
        <div class="flex items-center gap-1 md:hidden">
          {{ $t("label.sort-by") }}
          <DropdownMenu class="dropdown-left btn-xs" v-model="mobileSortField" :options="sortOptions" />
          <button
            class="btn btn-square btn-ghost btn-xs"
            @click="direction *= -1"
            :aria-label="$t('label.sort-direction')"
          >
            <mdi:arrow-up :class="direction > 0 ? '' : 'rotate-180'" />
          </button>
        </div>
        <div class="join max-md:hidden">
          <button
            class="icon-btn btn join-item btn-xs md:btn-sm"
            :class="statMode === 'chart' ? 'btn-active' : 'btn-ghost'"
            @click="statMode = 'chart'"
          >
            <mdi:chart-bar />
          </button>
          <button
            class="icon-btn btn join-item btn-xs md:btn-sm"
            :class="statMode === 'progress' ? 'btn-active' : 'btn-ghost'"
            @click="statMode = 'progress'"
          >
            <mdi:poll class="scale-x-[-1] rotate-90" />
          </button>
        </div>
      </div>
    </div>
    <div class="rounded-box border-base-content/10 overflow-x-auto border">
      <table class="table-md md:table-lg table-zebra table">
        <thead class="max-md:hidden">
          <tr :data-direction="direction > 0 ? 'asc' : 'desc'">
            <th
              v-for="(value, key) in fields"
              :key="key"
              @click.prevent="sort(key)"
              :class="[value.customClass, { 'selected-sort': key === sortField }]"
              v-show="isVisible(key)"
            >
              <a class="inline-flex cursor-pointer gap-2 text-sm uppercase">
                <span>{{ $t(isMobile && value.mobileLabel ? value.mobileLabel : value.label) }}</span>
                <span class="h-4" data-icon>
                  <mdi:arrow-up />
                </span>
              </a>
            </th>
          </tr>
        </thead>
        <tbody class="bg-base-300/30">
          <template v-if="!ready">
            <tr v-for="i in skeletonRows" :key="`skeleton-${i}`" role="status" class="animate-pulse">
              <td v-if="isVisible('name')" class="max-w-80 max-md:max-w-none">
                <div class="flex items-center gap-2">
                  <div class="bg-base-content/50 size-6 shrink-0 rounded-full opacity-50"></div>
                  <div class="bg-base-content/50 h-3 w-40 max-w-full rounded-full opacity-50"></div>
                  <span class="sr-only">Loading...</span>
                </div>
              </td>
              <td v-if="isVisible('host')">
                <div class="bg-base-content/50 h-3 w-20 rounded-full opacity-50"></div>
              </td>
              <td v-if="isVisible('state')">
                <div class="bg-base-content/50 h-3 w-16 rounded-full opacity-50"></div>
              </td>
              <td v-if="isVisible('created')">
                <div class="bg-base-content/50 h-3 w-24 rounded-full opacity-50"></div>
              </td>
              <td v-if="isVisible('cpu')">
                <div class="bg-base-content/50 h-3 w-full rounded-full opacity-50"></div>
              </td>
              <td v-if="isVisible('mem')">
                <div class="bg-base-content/50 h-3 w-full rounded-full opacity-50"></div>
              </td>
            </tr>
          </template>
          <tr
            v-else
            v-for="container in paginated"
            :key="container.id"
            v-memo="[
              container.id,
              container.state,
              container.health,
              statMode,
              isMobile,
              showAppIcons,
              dismissedLinkHint,
            ]"
            class="hover:bg-base-100/80!"
          >
            <td v-if="isVisible('name')" class="max-w-80 max-md:max-w-none">
              <div class="flex items-center gap-2 max-md:items-start">
                <ContainerIcon
                  :state="container.state"
                  :health="container.health"
                  :slug="container.icon"
                  class="size-6 max-md:mt-0.5"
                />
                <div class="min-w-0 flex-1">
                  <div class="flex items-baseline gap-2">
                    <router-link
                      class="min-w-0 flex-1 truncate"
                      :to="{ name: '/container/[id]', params: { id: container.id } }"
                      :title="container.name"
                    >
                      {{ container.name }}
                    </router-link>
                    <ContainerLink :container="container" />
                    <ContainerLinkHint :container="container" />
                    <RelativeTime
                      v-if="isMobile"
                      :date="container.created"
                      class="text-base-content/50 shrink-0 text-xs"
                    />
                  </div>
                  <div v-if="container.customGroup" class="text-base-content/50 truncate text-xs">
                    {{ container.customGroup }}
                  </div>
                  <div v-if="isMobile && container.state === 'running'" class="mt-1.5 flex items-center gap-1.5">
                    <ContainerStatCell
                      :container="container"
                      type="cpu"
                      :host="hosts[container.host]"
                      :mode="statMode"
                    />
                    <ContainerStatCell
                      :container="container"
                      type="mem"
                      :host="hosts[container.host]"
                      :mode="statMode"
                    />
                  </div>
                </div>
              </div>
            </td>
            <td v-if="isVisible('host')">{{ container.hostLabel }}</td>
            <td v-if="isVisible('state')">{{ container.health ?? container.state }}</td>
            <td v-if="isVisible('created')">
              <RelativeTime :date="container.created" />
            </td>
            <td v-if="isVisible('cpu')">
              <ContainerStatCell :container="container" type="cpu" :host="hosts[container.host]" :mode="statMode" />
            </td>
            <td v-if="isVisible('mem')">
              <ContainerStatCell :container="container" type="mem" :host="hosts[container.host]" :mode="statMode" />
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div class="p-4 text-center">
      <nav class="join" v-if="ready && isPaginated && totalPages <= 15">
        <input
          class="btn btn-square join-item"
          type="radio"
          v-model="currentPage"
          :aria-label="`${i}`"
          :value="i"
          v-for="i in totalPages"
        />
      </nav>
      <DropdownMenu
        v-else-if="ready && isPaginated"
        class="btn-sm"
        v-model="currentPage"
        :options="Array.from({ length: totalPages }, (_, i) => ({ label: `${i + 1}`, value: i + 1 }))"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { Container } from "@/models/Container";
import { showAppIcons } from "@/stores/settings";
import { toRefs } from "@vueuse/core";

const { t } = useI18n();
const { hosts } = useHosts();
const selectedHost = ref(null);

const fields: Record<
  string,
  {
    label: string;
    mobileLabel?: string;
    sortFunc: (a: Container, b: Container) => number;
    mobileVisible: boolean;
    customClass?: string;
  }
> = {
  name: {
    label: "label.container-name",
    mobileLabel: "label.name",
    sortFunc: (a: Container, b: Container) =>
      (a.name.localeCompare(b.name) || (a.customGroup ?? "").localeCompare(b.customGroup ?? "")) * direction.value,
    mobileVisible: true,
  },
  host: {
    label: "label.host",
    sortFunc: (a: Container, b: Container) => a.hostLabel.localeCompare(b.hostLabel) * direction.value,
    mobileVisible: false,
    customClass: "w-1",
  },
  state: {
    label: "label.status",
    sortFunc: (a: Container, b: Container) =>
      (a.health ?? a.state).localeCompare(b.health ?? b.state) * direction.value,
    mobileVisible: false,
    customClass: "w-1",
  },
  created: {
    label: "label.created",
    sortFunc: (a: Container, b: Container) => (a.created.getTime() - b.created.getTime()) * direction.value,
    mobileVisible: false,
    customClass: "w-1",
  },
  cpu: {
    label: "label.avg-cpu",
    mobileLabel: "label.cpu",
    sortFunc: (a: Container, b: Container) => (a.movingAverage.cpu - b.movingAverage.cpu) * direction.value,
    mobileVisible: false,
    customClass: "min-w-48 max-md:min-w-0",
  },
  mem: {
    label: "label.avg-mem",
    mobileLabel: "label.mem",
    sortFunc: (a: Container, b: Container) =>
      (a.movingAverage.memoryUsage - b.movingAverage.memoryUsage) * direction.value,
    mobileVisible: false,
    customClass: "min-w-48 max-md:min-w-0",
  },
};

const { containers } = defineProps<{
  containers: Container[];
}>();

const { ready } = storeToRefs(useContainerStore());
type keys = keyof typeof fields;

const statMode = useStorage<"chart" | "progress">("DOZZLE_TABLE_STAT_MODE", "chart");
const perPage = useStorage("DOZZLE_TABLE_PAGE_SIZE", 15);
const pageSizes = [15, 30, 50, 100];

const storage = useStorage<{ column: keys; direction: 1 | -1 }>("DOZZLE_TABLE_CONTAINERS_SORT", {
  column: "created" as keys,
  direction: -1 as 1 | -1,
});
const { column: sortField, direction } = toRefs(storage.value);
const counter = useInterval(10000);
const filteredContainers = computed(() =>
  containers.filter((c) => selectedHost.value === null || c.host === selectedHost.value),
);
const sortedContainers = computedWithControl(
  () => [filteredContainers.value.length, sortField.value, direction.value, counter.value],
  () => filteredContainers.value.sort((a, b) => fields[sortField.value].sortFunc(a, b)),
);

const skeletonRows = computed(() => Math.min(perPage.value, 5));
const totalPages = computed(() => Math.ceil(sortedContainers.value.length / perPage.value));
const isPaginated = computed(() => totalPages.value > 1);
const currentPage = ref(1);
watch(perPage, () => (currentPage.value = 1));
const paginated = computed(() => {
  const start = (currentPage.value - 1) * perPage.value;
  const end = start + perPage.value;

  return sortedContainers.value.slice(start, end);
});

const sortOptions = computed(() =>
  Object.entries(fields)
    .filter(([key]) => key !== "host" || Object.keys(hosts.value).length > 1)
    .map(([key, value]) => ({ label: t(value.mobileLabel ?? value.label), value: key })),
);

const mobileSortField = computed({
  get: () => sortField.value,
  set: (field: keys) => {
    if (field !== sortField.value) sort(field);
  },
});

function sort(field: keys) {
  if (sortField.value === field) {
    direction.value *= -1;
  } else {
    sortField.value = field;
    direction.value = 1;
  }
}
function isVisible(field: keys) {
  return fields[field].mobileVisible || !isMobile.value;
}
</script>

<style scoped>
@reference "@/main.css";

[data-icon] {
  display: none;
  transition: transform 0.2s ease-in-out;
  [data-direction="desc"] & {
    transform: rotate(180deg);
  }
}

th {
  @apply border-base-200 border-b-2;
  &.selected-sort {
    font-weight: bold;
    @apply border-primary;
    [data-icon] {
      display: inline-block;
    }
  }
}

tbody td {
  white-space: nowrap;
}

a {
  @apply hover:text-primary;
}
</style>
