<template>
  <PageWithLinks>
    <section>
      <!-- Header -->
      <div class="mb-8">
        <h2 class="text-2xl font-bold">Notifications</h2>
        <p class="text-base-content/60">Configure where and when to receive alerts</p>
      </div>

      <!-- Destinations Section -->
      <div class="mb-8">
        <h3 class="text-base-content/60 mb-4 font-semibold tracking-wide uppercase">Destinations</h3>
        <div class="flex flex-wrap gap-4">
          <DestinationCard v-for="dest in dispatchers" :key="dest.id" :destination="dest" class="w-full md:w-72" />
          <!-- Add Destination Card -->
          <button
            class="card card-border border-base-content/30 hover:border-base-content/50 w-full cursor-pointer border-dashed transition-colors md:w-72"
            @click="openAddDestination"
          >
            <div class="card-body items-center justify-center gap-1 p-4">
              <mdi:plus class="text-2xl" />
              <span class="text-base-content/60 text-sm">Add destination</span>
            </div>
          </button>
        </div>
      </div>

      <!-- Alerts Section -->
      <div>
        <div class="mb-4 flex items-center justify-between">
          <h3 class="text-base-content/60 font-semibold tracking-wide uppercase">Alerts</h3>
          <button class="btn btn-ghost text-primary" @click="openCreateAlert">
            <mdi:plus />
            Add
          </button>
        </div>

        <!-- Filter Tabs -->
        <div class="tabs tabs-box mb-6">
          <button class="tab" :class="{ 'tab-active': filter === 'all' }" @click="filter = 'all'">
            All ({{ alerts.length }})
          </button>
          <button class="tab" :class="{ 'tab-active': filter === 'enabled' }" @click="filter = 'enabled'">
            Enabled ({{ enabledCount }})
          </button>
          <button class="tab" :class="{ 'tab-active': filter === 'paused' }" @click="filter = 'paused'">
            Paused ({{ pausedCount }})
          </button>
        </div>

        <!-- Alerts List -->
        <div v-if="!alerts.length" class="text-base-content/60 py-4">
          No alerts configured yet. Create one to get started.
        </div>
        <div v-else class="space-y-4">
          <AlertCard v-for="alert in filteredAlerts" :key="alert.id" :alert="alert" />
        </div>
      </div>
    </section>
  </PageWithLinks>
</template>

<script lang="ts" setup>
import { useQuery } from "@urql/vue";
import { GetNotificationRulesDocument, GetDispatchersDocument, type NotificationRule } from "@/types/graphql";
import AlertForm from "@/components/Notification/AlertForm.vue";
import DestinationForm from "@/components/Notification/DestinationForm.vue";
import DestinationCard from "@/components/Notification/DestinationCard.vue";

const showDrawer = useDrawer();

// GraphQL queries
const alertsQuery = useQuery({ query: GetNotificationRulesDocument });
const dispatchersQuery = useQuery({ query: GetDispatchersDocument });

// Computed data from queries
const alerts = computed(() => alertsQuery.data.value?.notificationRules ?? []);
const dispatchers = computed(() => dispatchersQuery.data.value?.dispatchers ?? []);

// Local state
const filter = ref<"all" | "enabled" | "paused">("all");

const enabledCount = computed(() => alerts.value.filter((a: NotificationRule) => a.enabled).length);
const pausedCount = computed(() => alerts.value.filter((a: NotificationRule) => !a.enabled).length);

const filteredAlerts = computed(() => {
  if (filter.value === "enabled") return alerts.value.filter((a: NotificationRule) => a.enabled);
  if (filter.value === "paused") return alerts.value.filter((a: NotificationRule) => !a.enabled);
  return alerts.value;
});

function openCreateAlert() {
  showDrawer(AlertForm, { onCreated: () => alertsQuery.executeQuery({ requestPolicy: "network-only" }) }, "lg");
}

function openAddDestination() {
  showDrawer(
    DestinationForm,
    { onCreated: () => dispatchersQuery.executeQuery({ requestPolicy: "network-only" }) },
    "md",
  );
}
</script>
