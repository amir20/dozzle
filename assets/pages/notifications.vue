<template>
  <PageWithLinks>
    <section>
      <!-- Header -->
      <div class="mb-8">
        <h2 class="text-2xl font-bold">{{ $t("notifications.title") }}</h2>
        <p class="text-base-content/60">{{ $t("notifications.description") }}</p>
      </div>

      <!-- Destinations Section -->
      <div class="mb-8">
        <h3 class="text-base-content/60 mb-4 font-semibold tracking-wide uppercase">
          {{ $t("notifications.destinations") }}
        </h3>
        <div class="flex flex-wrap gap-4">
          <DestinationCard
            v-for="dest in dispatchers"
            :key="dest.id"
            :destination="dest"
            :on-updated="fetchDispatchers"
            :existing-dispatchers="dispatchers"
            class="w-full md:w-72"
          />
          <!-- Add Destination Card -->
          <button
            class="card card-border border-base-content/30 hover:border-base-content/50 w-full cursor-pointer border-dashed transition-colors md:w-72"
            @click="openAddDestination"
          >
            <div class="card-body items-center justify-center gap-1 p-4">
              <mdi:plus class="text-2xl" />
              <span class="text-base-content/60 text-sm">{{ $t("notifications.add-destination") }}</span>
            </div>
          </button>
        </div>
      </div>

      <!-- Alerts Section -->
      <div>
        <div class="mb-4 flex items-center justify-between">
          <h3 class="text-base-content/60 font-semibold tracking-wide uppercase">{{ $t("notifications.alerts") }}</h3>
          <button class="btn btn-ghost text-primary" @click="openCreateAlert">
            <mdi:plus />
            {{ $t("notifications.add") }}
          </button>
        </div>

        <!-- Filter Tabs -->
        <div class="tabs tabs-box mb-6">
          <button class="tab" :class="{ 'tab-active': filter === 'all' }" @click="filter = 'all'">
            {{ $t("notifications.filter.all", { count: alerts.length }) }}
          </button>
          <button class="tab" :class="{ 'tab-active': filter === 'enabled' }" @click="filter = 'enabled'">
            {{ $t("notifications.filter.enabled", { count: enabledCount }) }}
          </button>
          <button class="tab" :class="{ 'tab-active': filter === 'paused' }" @click="filter = 'paused'">
            {{ $t("notifications.filter.paused", { count: pausedCount }) }}
          </button>
        </div>

        <!-- Alerts List -->
        <div v-if="!alerts.length" class="text-base-content/60 py-4">
          {{ $t("notifications.no-alerts") }}
        </div>
        <div v-else class="space-y-4">
          <AlertCard v-for="alert in filteredAlerts" :key="alert.id" :alert="alert" :on-updated="fetchAlerts" />
        </div>
      </div>
    </section>
  </PageWithLinks>
</template>

<script lang="ts" setup>
import type { NotificationRule, Dispatcher } from "@/types/notifications";
import AlertForm from "@/components/Notification/AlertForm.vue";
import DestinationForm from "@/components/Notification/DestinationForm.vue";
import DestinationCard from "@/components/Notification/DestinationCard.vue";

const showDrawer = useDrawer();
const route = useRoute();

// State
const alerts = ref<NotificationRule[]>([]);
const dispatchers = ref<Dispatcher[]>([]);

async function fetchAlerts() {
  const res = await fetch(withBase("/api/notifications/rules"));
  alerts.value = await res.json();
}

async function fetchDispatchers() {
  const res = await fetch(withBase("/api/notifications/dispatchers"));
  dispatchers.value = await res.json();
}

onMounted(() => {
  fetchAlerts();
  fetchDispatchers();
});

// Handle newCloudLink query param
watch(
  () => [route.query.newCloudLink, dispatchers.value] as const,
  ([newCloudLink, data]) => {
    if (newCloudLink && data?.length) {
      const id = Number(newCloudLink);
      const destination = dispatchers.value.find((d) => d.id === id);
      if (destination) {
        showDrawer(
          DestinationForm,
          {
            destination,
            onCreated: fetchDispatchers,
            existingDispatchers: dispatchers.value,
          },
          "md",
        );
      }
    }
  },
  { immediate: true },
);

// Local state
const filter = ref<"all" | "enabled" | "paused">("all");

const enabledCount = computed(() => alerts.value.filter((a) => a.enabled).length);
const pausedCount = computed(() => alerts.value.filter((a) => !a.enabled).length);

const filteredAlerts = computed(() => {
  if (filter.value === "enabled") return alerts.value.filter((a) => a.enabled);
  if (filter.value === "paused") return alerts.value.filter((a) => !a.enabled);
  return alerts.value;
});

function openCreateAlert() {
  showDrawer(AlertForm, { onCreated: fetchAlerts }, "lg");
}

function openAddDestination() {
  showDrawer(
    DestinationForm,
    {
      onCreated: fetchDispatchers,
      existingDispatchers: dispatchers.value,
    },
    "md",
  );
}
</script>
