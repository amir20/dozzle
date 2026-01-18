<template>
  <PageWithLinks>
    <section>
      <div class="mb-6 flex items-center justify-between">
        <div>
          <h2 class="text-2xl font-bold">Notifications</h2>
          <p class="text-base-content/60">Manage your log event subscriptions</p>
        </div>
        <button class="btn btn-primary" @click="openCreateAlert">
          <mdi:plus />
          Create Alert
        </button>
      </div>

      <!-- Filter Tabs -->
      <div class="tabs tabs-box mb-6">
        <button class="tab" :class="{ 'tab-active': filter === 'all' }" @click="filter = 'all'">
          All ({{ subscriptions.length }})
        </button>
        <button class="tab" :class="{ 'tab-active': filter === 'enabled' }" @click="filter = 'enabled'">
          Enabled ({{ enabledCount }})
        </button>
        <button class="tab" :class="{ 'tab-active': filter === 'paused' }" @click="filter = 'paused'">
          Paused ({{ pausedCount }})
        </button>
      </div>

      <!-- Alerts List -->
      <div v-if="isLoading" class="flex justify-center py-8">
        <span class="loading loading-spinner loading-md"></span>
      </div>
      <div v-else-if="!subscriptions.length" class="text-base-content/60 py-4">
        No alerts configured yet. Create one to get started.
      </div>
      <div v-else class="space-y-4">
        <div
          v-for="sub in filteredSubscriptions"
          :key="sub.id"
          class="card bg-base-200 shadow-sm"
          :class="{ 'opacity-60': !sub.enabled }"
        >
          <div class="card-body gap-4 p-5">
            <!-- Header -->
            <div class="flex items-start justify-between">
              <div class="flex items-center gap-2">
                <h4 class="text-lg font-semibold">{{ sub.name }}</h4>
                <span v-if="!sub.enabled" class="badge badge-ghost badge-sm">Paused</span>
              </div>
              <input
                type="checkbox"
                class="toggle toggle-primary"
                :checked="sub.enabled"
                @change="toggleEnabled(sub)"
              />
            </div>

            <!-- Expressions -->
            <div class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
              <span>Container Filter</span>
              <code class="bg-base-300 p-4 py-0.5 font-mono">{{ sub.containerExpression }}</code>
              <span>Log filter</span>
              <code class="bg-base-300 p-4 py-0.5 font-mono">{{ sub.logExpression }}</code>
            </div>

            <!-- Footer -->
            <div class="border-base-300 text-base-content/80 flex items-center justify-between border-t pt-3 text-sm">
              <div class="flex items-center gap-4">
                <span class="flex items-center gap-1">
                  <mdi:package-variant-closed class="text-base" />
                  {{ sub.triggeredContainers }} containers
                </span>
                <span class="flex items-center gap-1">
                  <mdi:bell-outline class="text-base" />
                  {{ sub.triggerCount }} triggered
                </span>
                <span v-if="sub.lastTriggeredAt" class="flex items-center gap-1">
                  <mdi:clock-outline class="text-base" />
                  Last: {{ formatTimeAgo(sub.lastTriggeredAt) }}
                </span>
              </div>
              <div class="flex items-center gap-1">
                <button class="btn btn-ghost btn-sm btn-square" @click="editSubscription(sub)">
                  <mdi:pencil-outline />
                </button>
                <button
                  class="btn btn-ghost btn-sm btn-square text-error"
                  @click="deleteSubscription(sub.id)"
                  :disabled="deletingId === sub.id"
                >
                  <span v-if="deletingId === sub.id" class="loading loading-spinner loading-xs"></span>
                  <mdi:delete-outline v-else />
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  </PageWithLinks>
</template>

<script lang="ts" setup>
import CreateAlert from "@/components/Notification/CreateAlert.vue";

interface Subscription {
  id: number;
  name: string;
  enabled: boolean;
  containerExpression: string;
  logExpression: string;
  triggerCount: number;
  triggeredContainers: number;
  lastTriggeredAt: string | null;
}

const showDrawer = useDrawer();

const subscriptions = ref<Subscription[]>([]);
const isLoading = ref(true);
const deletingId = ref<number | null>(null);
const filter = ref<"all" | "enabled" | "paused">("all");

const enabledCount = computed(() => subscriptions.value.filter((s) => s.enabled).length);
const pausedCount = computed(() => subscriptions.value.filter((s) => !s.enabled).length);

const filteredSubscriptions = computed(() => {
  if (filter.value === "enabled") return subscriptions.value.filter((s) => s.enabled);
  if (filter.value === "paused") return subscriptions.value.filter((s) => !s.enabled);
  return subscriptions.value;
});

function formatTimeAgo(dateStr: string): string {
  const date = new Date(dateStr);
  if (date.getFullYear() === 0) return "-";
  return toRelativeTime(date, undefined);
}

async function fetchSubscriptions() {
  try {
    const response = await fetch(withBase("/api/notifications/subscriptions"));
    if (response.ok) {
      subscriptions.value = await response.json();
    }
  } finally {
    isLoading.value = false;
  }
}

async function deleteSubscription(id: number) {
  deletingId.value = id;
  try {
    const response = await fetch(withBase(`/api/notifications/subscriptions/${id}`), {
      method: "DELETE",
    });
    if (response.ok) {
      subscriptions.value = subscriptions.value.filter((s) => s.id !== id);
    }
  } finally {
    deletingId.value = null;
  }
}

async function toggleEnabled(sub: Subscription) {
  const newEnabled = !sub.enabled;
  sub.enabled = newEnabled; // Optimistic update
  try {
    await fetch(withBase(`/api/notifications/subscriptions/${sub.id}`), {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enabled: newEnabled }),
    });
  } catch {
    sub.enabled = !newEnabled; // Revert on error
  }
}

function editSubscription(sub: Subscription) {
  // TODO: Open edit drawer
  console.log("Edit subscription", sub.id);
}

function openCreateAlert() {
  showDrawer(CreateAlert, { onCreated: fetchSubscriptions }, "lg");
}

fetchSubscriptions();
</script>
