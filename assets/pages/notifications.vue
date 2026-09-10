<template>
  <PageWithLinks>
    <section>
      <div class="mb-6">
        <h2 class="text-2xl font-bold">{{ $t("notifications.title") }}</h2>
        <p class="text-base-content/60">{{ $t("notifications.description") }}</p>
      </div>

      <!--
        Three surfaces, one page. Stacked they made a page nobody reaches the
        bottom of, and the rules you came to edit sat between a destination
        picker and a history you were not looking for. Tabs also give the bell
        somewhere to point: a new alert opens the history directly.
      -->
      <div role="tablist" class="tabs tabs-border border-base-content/10 mb-6 border-b">
        <button
          v-for="entry in tabs"
          :key="entry.id"
          role="tab"
          class="tab gap-2"
          :class="{ 'tab-active': tab === entry.id }"
          @click="selectTab(entry.id)"
        >
          {{ entry.label }}
          <span v-if="entry.count !== undefined" class="text-base-content/40 font-mono text-xs">
            {{ entry.count }}
          </span>
          <span v-if="entry.dot" class="bg-warning size-1.5 rounded-full"></span>
        </button>
      </div>

      <!-- ACTIVITY -->
      <AlertHistory v-if="tab === 'activity'" />

      <!-- ALERTS -->
      <template v-else-if="tab === 'alerts'">
        <div class="mb-4 flex flex-wrap items-center gap-3">
          <!-- Nothing to filter until there is more than one alert. -->
          <div v-if="alerts.length > 1" role="tablist" class="tabs tabs-box tabs-xs">
            <button
              v-for="option in filters"
              :key="option.id"
              role="tab"
              class="tab"
              :class="{ 'tab-active': filter === option.id }"
              @click="filter = option.id"
            >
              {{ option.label }}
            </button>
          </div>
          <button class="btn btn-primary btn-sm ml-auto" @click="openCreateAlert">
            <mdi:plus class="size-4" />
            {{ $t("notifications.add-alert") }}
          </button>
        </div>

        <div class="space-y-4">
          <p v-if="!alerts.length" class="text-base-content/60 text-sm">{{ $t("notifications.no-alerts") }}</p>
          <p v-else-if="!filteredAlerts.length" class="text-base-content/60 text-sm">
            {{ $t("notifications.no-alerts-in-filter") }}
          </p>
          <AlertCard
            v-for="alert in filteredAlerts"
            :key="alert.id"
            :alert="alert"
            :dispatchers="dispatchers"
            :on-updated="fetchAlerts"
            :highlight="alert.id === highlightId"
          />
        </div>
      </template>

      <!-- DESTINATIONS -->
      <template v-else>
        <div class="mb-4 flex flex-wrap items-center gap-3">
          <p v-if="!dispatchers.length" class="text-base-content/60 text-sm">
            {{ $t("notifications.empty-state.description") }}
          </p>
          <button class="btn btn-primary btn-sm ml-auto" @click="openAddDestination">
            <mdi:plus class="size-4" />
            {{ $t("notifications.add-destination") }}
          </button>
        </div>

        <div class="flex flex-wrap gap-4">
          <DestinationCard
            v-for="dest in dispatchers"
            :key="dest.id"
            :destination="dest"
            :on-updated="fetchAll"
            :existing-dispatchers="dispatchers"
            :used-by-count="alertsPerDispatcher.get(dest.id) ?? 0"
            class="w-full md:w-72"
          />
        </div>
      </template>
    </section>
  </PageWithLinks>
</template>

<script lang="ts" setup>
import type { NotificationRule, Dispatcher } from "@/types/notifications";
import AlertForm from "@/components/notifications/AlertForm.vue";
import DestinationForm from "@/components/notifications/DestinationForm.vue";

const { t } = useI18n();
const showDrawer = useDrawer();
const router = useRouter();
const route = useRoute();

// State
const alerts = ref<NotificationRule[]>([]);
const dispatchers = ref<Dispatcher[]>([]);

type Tab = "activity" | "alerts" | "destinations";
const { linked: cloudLinked } = useCloudSurface();
const { unseen: unseenAlerts, markAlertsSeen } = useRecentAlerts();

// What happened, then the rules that made it happen, then where it goes. An
// instance with no cloud link has no history to open, so it lands on the rules
// instead of on a muted line explaining an absence.
const defaultTab = computed<Tab>(() => (cloudLinked.value ? "activity" : "alerts"));

// The tab lives in the URL so a bookmark and the back button land where the
// reader left off. Only when it differs from the default, so the plain
// /notifications link the bell uses stays plain.
const tab = computed<Tab>(() => {
  const value = route.query.tab;
  return value === "activity" || value === "alerts" || value === "destinations" ? value : defaultTab.value;
});

function queryForTab(next: Tab) {
  return next === defaultTab.value ? undefined : next;
}

function selectTab(next: Tab) {
  router.replace({ query: { ...route.query, tab: queryForTab(next) } });
}

const tabs = computed(() => [
  { id: "activity" as const, label: t("notifications.history.title"), dot: unseenAlerts.value },
  { id: "alerts" as const, label: t("notifications.alerts"), count: alerts.value.length },
  { id: "destinations" as const, label: t("notifications.destinations"), count: dispatchers.value.length },
]);

// Reading the history is what clears the bell, so it happens on arrival and
// again when the rows land — the fetch usually finishes after the tab opens.
watchEffect(() => {
  if (tab.value === "activity" && unseenAlerts.value) markAlertsSeen();
});

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

const highlightId = ref<number | null>(null);
const { showToast } = useToast();

function consumeHighlight(value: unknown) {
  if (typeof value !== "string" || !value) return false;
  const parsed = Number.parseInt(value, 10);
  if (!Number.isFinite(parsed)) return false;
  highlightId.value = parsed;
  router.replace({ query: { tab: queryForTab("alerts") } });
  showToast(
    {
      type: "info",
      message: t("notifications.default-alert-created"),
    },
    { expire: 8000 },
  );
  return true;
}

function consumeAction(action: unknown) {
  if (action !== "create-alert") return;
  router.replace({ query: { tab: queryForTab("alerts") } });
  openCreateAlertPrefilled();
}

onMounted(async () => {
  await fetchAll();
  const hash = window.location.hash;
  if (hash === "#cloudLinked") {
    router.replace({ hash: "" });
  }

  if (!consumeHighlight(route.query.highlight)) {
    consumeAction(route.query.action);
  }
});

watch(
  () => route.query.highlight,
  (value) => consumeHighlight(value),
);

watch(
  () => route.query.action,
  (action) => consumeAction(action),
);

// Local state
const filter = ref<"all" | "enabled" | "paused">("all");

const enabledCount = computed(() => alerts.value.filter((a) => a.enabled).length);
const pausedCount = computed(() => alerts.value.filter((a) => !a.enabled).length);

const filters = computed(() => [
  { id: "all" as const, label: t("notifications.filter.all", { count: alerts.value.length }) },
  { id: "enabled" as const, label: t("notifications.filter.enabled", { count: enabledCount.value }) },
  { id: "paused" as const, label: t("notifications.filter.paused", { count: pausedCount.value }) },
]);

// Deleting a destination orphans the alerts pointing at it, so each card shows its usage.
const alertsPerDispatcher = computed(() => {
  const counts = new Map<number, number>();
  for (const alert of alerts.value) {
    if (!alert.dispatcher) continue;
    counts.set(alert.dispatcher.id, (counts.get(alert.dispatcher.id) ?? 0) + 1);
  }
  return counts;
});

const filteredAlerts = computed(() => {
  if (filter.value === "enabled") return alerts.value.filter((a) => a.enabled);
  if (filter.value === "paused") return alerts.value.filter((a) => !a.enabled);
  return alerts.value;
});

function openCreateAlert() {
  showDrawer(AlertForm, { onCreated: fetchAlerts }, "lg");
}

function openCreateAlertPrefilled() {
  const cloudDispatcher = dispatchers.value.find((d) => d.type === "cloud");
  showDrawer(
    AlertForm,
    {
      onCreated: fetchAlerts,
      prefill: {
        name: t("notifications.prefill-name"),
        alertType: "log" as const,
        logExpression: t("notifications.prefill-expression"),
        ...(cloudDispatcher ? { dispatcherId: cloudDispatcher.id } : {}),
      },
    },
    "lg",
  );
}

function openAddDestination() {
  showDrawer(
    DestinationForm,
    {
      onCreated: fetchDispatchers,
    },
    "md",
  );
}
</script>
