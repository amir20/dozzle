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
          <DestinationCard
            v-for="dest in dispatchers"
            :key="dest.id"
            :destination="dest"
            @edit="editDestination"
            @delete="handleDeleteDispatcher"
          />
          <!-- Add Destination Card -->
          <button
            class="card card-border border-base-content/30 hover:border-base-content/50 w-72 cursor-pointer border-dashed transition-colors"
            @click="openAddDestination"
          >
            <div class="card-body items-center justify-center gap-1">
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
          <div
            v-for="alert in filteredAlerts"
            :key="alert.id"
            class="card bg-base-100 shadow-sm"
            :class="{ 'opacity-60': !alert.enabled }"
          >
            <div class="card-body gap-4 p-5">
              <!-- Header -->
              <div class="flex items-start justify-between">
                <div class="flex items-center gap-2">
                  <h4 class="text-lg font-semibold">{{ alert.name }}</h4>
                  <span v-if="!alert.enabled" class="badge badge-warning badge-sm">Paused</span>
                </div>
                <input
                  type="checkbox"
                  class="toggle toggle-primary"
                  :checked="alert.enabled"
                  @change="toggleEnabled(alert)"
                />
              </div>

              <!-- Expressions -->
              <div class="text-base-content/80 grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
                <span>Containers</span>
                <code class="bg-base-200 text-base-content rounded px-2 py-0.5 font-mono">{{
                  alert.containerExpression
                }}</code>
                <span>Log filter</span>
                <code class="bg-base-200 text-base-content rounded px-2 py-0.5 font-mono">{{
                  alert.logExpression
                }}</code>
                <span>Destination</span>
                <span class="flex items-center gap-1.5">
                  <mdi:webhook v-if="alert.dispatcher.type === 'webhook'" />
                  <mdi:cloud v-else />
                  {{ alert.dispatcher.name }}
                </span>
              </div>

              <!-- Footer -->
              <div
                class="border-base-content/10 text-base-content/80 flex items-center justify-between border-t pt-3 text-sm"
              >
                <div class="flex items-center gap-4">
                  <span class="flex items-center gap-1">
                    <mdi:package-variant-closed class="text-base" />
                    {{ alert.triggeredContainers }} containers
                  </span>
                  <span class="flex items-center gap-1">
                    <mdi:bell-outline class="text-base" />
                    {{ alert.triggerCount }} triggered
                  </span>
                  <span v-if="alert.lastTriggeredAt" class="flex items-center gap-1">
                    <mdi:clock-outline class="text-base" />
                    Last: {{ formatTimeAgo(alert.lastTriggeredAt) }}
                  </span>
                </div>
                <div class="flex items-center gap-1">
                  <button class="btn btn-ghost btn-square" @click="editAlert(alert)">
                    <mdi:pencil-outline />
                  </button>
                  <button
                    class="btn btn-ghost btn-square text-error"
                    @click="handleDeleteAlert(alert.id)"
                    :disabled="deletingId === alert.id"
                  >
                    <span v-if="deletingId === alert.id" class="loading loading-spinner loading-xs"></span>
                    <mdi:trash-can-outline v-else />
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  </PageWithLinks>
</template>

<script lang="ts" setup>
import { useQuery, useMutation } from "@urql/vue";
import {
  GetNotificationRulesDocument,
  GetDispatchersDocument,
  DeleteNotificationRuleDocument,
  UpdateNotificationRuleDocument,
  DeleteDispatcherDocument,
  type NotificationRule,
  type Dispatcher,
} from "@/types/graphql";
import AlertForm from "@/components/Notification/AlertForm.vue";
import DestinationForm from "@/components/Notification/DestinationForm.vue";
import DestinationCard from "@/components/Notification/DestinationCard.vue";

const showDrawer = useDrawer();

// GraphQL queries
const alertsQuery = useQuery({ query: GetNotificationRulesDocument });
const dispatchersQuery = useQuery({ query: GetDispatchersDocument });

// GraphQL mutations
const deleteAlertMutation = useMutation(DeleteNotificationRuleDocument);
const updateAlertMutation = useMutation(UpdateNotificationRuleDocument);
const deleteDispatcherMutation = useMutation(DeleteDispatcherDocument);

// Computed data from queries
const alerts = computed(() => alertsQuery.data.value?.notificationRules ?? []);
const dispatchers = computed(() => dispatchersQuery.data.value?.dispatchers ?? []);

// Local state
const deletingId = ref<number | null>(null);
const filter = ref<"all" | "enabled" | "paused">("all");

const enabledCount = computed(() => alerts.value.filter((a: NotificationRule) => a.enabled).length);
const pausedCount = computed(() => alerts.value.filter((a: NotificationRule) => !a.enabled).length);

const filteredAlerts = computed(() => {
  if (filter.value === "enabled") return alerts.value.filter((a: NotificationRule) => a.enabled);
  if (filter.value === "paused") return alerts.value.filter((a: NotificationRule) => !a.enabled);
  return alerts.value;
});

function formatTimeAgo(dateStr: string): string {
  const date = new Date(dateStr);
  if (date.getFullYear() === 0) return "-";
  return toRelativeTime(date, undefined);
}

async function handleDeleteAlert(id: number) {
  deletingId.value = id;
  try {
    await deleteAlertMutation.executeMutation({ id });
    alertsQuery.executeQuery({ requestPolicy: "network-only" });
  } finally {
    deletingId.value = null;
  }
}

async function toggleEnabled(alert: NotificationRule) {
  await updateAlertMutation.executeMutation({
    id: alert.id,
    input: { enabled: !alert.enabled },
  });
  alertsQuery.executeQuery({ requestPolicy: "network-only" });
}

function editAlert(alert: NotificationRule) {
  showDrawer(AlertForm, { alert, onCreated: () => alertsQuery.executeQuery({ requestPolicy: "network-only" }) }, "lg");
}

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

function editDestination(destination: Dispatcher) {
  showDrawer(
    DestinationForm,
    { destination, onCreated: () => dispatchersQuery.executeQuery({ requestPolicy: "network-only" }) },
    "md",
  );
}

async function handleDeleteDispatcher(destination: Dispatcher) {
  await deleteDispatcherMutation.executeMutation({ id: destination.id });
  dispatchersQuery.executeQuery({ requestPolicy: "network-only" });
}
</script>
