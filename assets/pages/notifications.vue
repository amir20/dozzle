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

        <!-- Empty state: two option cards -->
        <template v-if="dispatchers.length === 0">
          <p class="text-base-content/60 mb-4 text-sm">{{ $t("notifications.empty-state.description") }}</p>
          <div class="flex flex-wrap gap-4">
            <!-- Dozzle Cloud card -->
            <button
              class="card card-border border-primary bg-primary/5 hover:bg-primary/10 w-full cursor-pointer transition-colors md:w-72"
              @click="handleCloudDestination"
            >
              <div class="card-body gap-2 p-5">
                <mdi:cloud-outline class="text-primary text-2xl" />
                <div class="text-left">
                  <div class="font-semibold">{{ $t("notifications.destination.dozzle-cloud") }}</div>
                  <div class="text-base-content/60 text-sm">{{ $t("notifications.empty-state.cloud-subtitle") }}</div>
                </div>
              </div>
            </button>
            <!-- HTTP Webhook card -->
            <button
              class="card card-border border-base-content/20 hover:border-base-content/40 w-full cursor-pointer transition-colors md:w-72"
              @click="openAddWebhook"
            >
              <div class="card-body gap-2 p-5">
                <mdi:webhook class="text-2xl" />
                <div class="text-left">
                  <div class="font-semibold">{{ $t("notifications.destination.http-webhook") }}</div>
                  <div class="text-base-content/60 text-sm">{{ $t("notifications.empty-state.webhook-subtitle") }}</div>
                </div>
              </div>
            </button>
          </div>
        </template>

        <!-- Has destinations: show cards + add button -->
        <template v-else>
          <div class="flex flex-wrap gap-4">
            <DestinationCard
              v-for="dest in dispatchers"
              :key="dest.id"
              :destination="dest"
              :on-updated="fetchAll"
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
        </template>
      </div>

      <!-- Alerts Section -->
      <div>
        <div class="mb-4">
          <h3 class="text-base-content/60 font-semibold tracking-wide uppercase">{{ $t("notifications.alerts") }}</h3>
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
        <div class="space-y-4">
          <AlertCard v-for="alert in filteredAlerts" :key="alert.id" :alert="alert" :on-updated="fetchAlerts" />
          <button
            class="card card-border border-base-content/30 hover:border-base-content/50 w-full cursor-pointer border-dashed transition-colors"
            @click="openCreateAlert"
          >
            <div class="card-body items-center justify-center gap-1 p-4">
              <mdi:plus class="text-2xl" />
              <span class="text-base-content/60 text-sm">{{ $t("notifications.add-alert") }}</span>
            </div>
          </button>
        </div>
      </div>
    </section>
  </PageWithLinks>
</template>

<script lang="ts" setup>
import type { NotificationRule, Dispatcher, CloudConfig } from "@/types/notifications";
import AlertForm from "@/components/Notification/AlertForm.vue";
import DestinationForm from "@/components/Notification/DestinationForm.vue";

const showDrawer = useDrawer();
const router = useRouter();

// State
const alerts = ref<NotificationRule[]>([]);
const dispatchers = ref<Dispatcher[]>([]);
const cloudConfig = ref<CloudConfig | null>(null);

async function fetchAlerts() {
  const res = await fetch(withBase("/api/notifications/rules"));
  alerts.value = await res.json();
}

async function fetchDispatchers() {
  const res = await fetch(withBase("/api/notifications/dispatchers"));
  dispatchers.value = await res.json();
}

async function fetchAll() {
  await Promise.all([fetchAlerts(), fetchDispatchers()]);
}

async function fetchCloudConfig() {
  try {
    const res = await fetch(withBase("/api/cloud/config"));
    if (res.ok) {
      cloudConfig.value = await res.json();
    }
  } catch {
    cloudConfig.value = null;
  }
}

// Handle cloudLinkSuccess hash param
onMounted(async () => {
  await Promise.all([fetchAll(), fetchCloudConfig()]);
  const hash = window.location.hash;
  if (hash.startsWith("#cloudLinkSuccess=")) {
    const id = Number(hash.replace("#cloudLinkSuccess=", ""));
    if (!isNaN(id)) {
      const destination = dispatchers.value.find((d) => d.id === id);
      if (destination) {
        showDrawer(
          DestinationForm,
          {
            destination,
            existingDispatchers: dispatchers.value,
            showLinkSuccess: true,
          },
          "md",
        );
      }
    }
    router.replace({ hash: "" });
  }
});

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

function handleCloudDestination() {
  if (cloudConfig.value?.linked) {
    // Already linked — open add destination with cloud pre-selected
    showDrawer(
      DestinationForm,
      {
        onCreated: fetchDispatchers,
        existingDispatchers: dispatchers.value,
        defaultType: "cloud" as const,
      },
      "md",
    );
  } else {
    // Not linked — start OAuth
    const callbackUrl = `${window.location.origin}${withBase("/")}`;
    window.location.href = `${__CLOUD_URL__}/link?appUrl=${encodeURIComponent(callbackUrl)}&from=notifications`;
  }
}

function openAddWebhook() {
  showDrawer(
    DestinationForm,
    {
      onCreated: fetchDispatchers,
      existingDispatchers: dispatchers.value,
      defaultType: "webhook" as const,
    },
    "md",
  );
}
</script>
