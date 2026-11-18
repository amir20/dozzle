<template>
  <PageWithLinks>
    <section>
      <div class="mb-6">
        <h2 class="text-2xl font-bold">{{ $t("notifications.title") }}</h2>
        <p class="text-base-content/60">{{ $t("notifications.description") }}</p>
      </div>

      <!-- The rules, then where they go. What they did is the cloud drawer's
           Alerts panel, which is where the bell points. -->
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
        </button>
      </div>

      <!-- ALERTS -->
      <template v-if="tab === 'alerts'">
        <!-- With nothing to list, the button is the whole page, so it sits under the
             sentence that explains it instead of alone at the far edge. -->
        <EmptyState v-if="!alerts.length" :title="$t('notifications.no-alerts')">
          <template #icon><mdi:bell-outline class="size-5" /></template>
          <template #actions>
            <button class="btn btn-primary btn-sm" @click="openCreateAlert">
              <mdi:plus class="size-4" />
              {{ $t("notifications.add-alert") }}
            </button>
          </template>
        </EmptyState>

        <div v-else class="mb-4 flex flex-wrap items-center gap-3">
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

        <div v-if="alerts.length" class="space-y-4">
          <p v-if="!filteredAlerts.length" class="text-base-content/60 text-sm">
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
        <EmptyState
          v-if="!dispatchers.length"
          :title="$t('notifications.empty-state.title')"
          :hint="$t('notifications.empty-state.description')"
        >
          <template #icon><mdi:send-outline class="size-5" /></template>
          <template #actions>
            <button class="btn btn-primary btn-sm" @click="openAddDestination">
              <mdi:plus class="size-4" />
              {{ $t("notifications.add-destination") }}
            </button>
          </template>
        </EmptyState>

        <div v-else class="mb-4 flex flex-wrap items-center gap-3">
          <button class="btn btn-primary btn-sm ml-auto" @click="openAddDestination">
            <mdi:plus class="size-4" />
            {{ $t("notifications.add-destination") }}
          </button>
        </div>

        <div
          v-if="dispatchers.length"
          class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border"
        >
          <DestinationCard
            v-for="dest in dispatchers"
            :key="dest.id"
            :destination="dest"
            :on-updated="fetchAll"
            :existing-dispatchers="dispatchers"
            :used-by="alertsPerDispatcher.get(dest.id) ?? []"
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

type Tab = "alerts" | "destinations";
const { linked: cloudLinked } = useCloudSurface();
const { openRail } = useCloudRail();

const DEFAULT_TAB: Tab = "alerts";

// The tab lives in the URL so a bookmark and the back button land where the
// reader left off. Only when it differs from the default, so a plain
// /notifications link stays plain.
const tab = computed<Tab>(() => (route.query.tab === "destinations" ? "destinations" : DEFAULT_TAB));

// ?tab=activity is the history this page used to hold: a bookmark, an older
// email, the phone's bell. The history is the drawer's Alerts panel now, so
// that is where it opens. Unlinked there is no history, and the rules show.
watch(
  () => route.query.tab,
  (value) => {
    if (value !== "activity") return;
    router.replace({ query: { ...route.query, tab: undefined } });
    if (cloudLinked.value) openRail("alerts");
  },
  { immediate: true },
);

function queryForTab(next: Tab) {
  return next === DEFAULT_TAB ? undefined : next;
}

function selectTab(next: Tab) {
  router.replace({ query: { ...route.query, tab: queryForTab(next) } });
}

const tabs = computed(() => [
  { id: "alerts" as const, label: t("notifications.alerts"), count: alerts.value.length },
  { id: "destinations" as const, label: t("notifications.destinations"), count: dispatchers.value.length },
]);

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

// Deleting a destination orphans the alerts pointing at it, so each row lists them.
const alertsPerDispatcher = computed(() => {
  const byId = new Map<number, NotificationRule[]>();
  for (const alert of alerts.value) {
    if (!alert.dispatcher) continue;
    const list = byId.get(alert.dispatcher.id) ?? [];
    list.push(alert);
    byId.set(alert.dispatcher.id, list);
  }
  return byId;
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
