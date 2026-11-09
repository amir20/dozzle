<template>
  <div
    class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border"
    :class="{ 'highlight-new': isHighlighted }"
    @animationend="isHighlighted = false"
  >
    <!-- Paused fades the rule, not the panel, so the toggle that brings it back keeps full contrast. -->
    <div class="flex items-start gap-3 p-4">
      <div
        class="flex size-8 shrink-0 items-center justify-center rounded-full"
        :class="alert.enabled ? 'bg-info/10 text-info' : 'bg-base-content/10 text-base-content/50'"
      >
        <mdi:chart-line v-if="alert.metricExpression" class="size-4" />
        <mdi:bell-ring-outline v-else-if="alert.eventExpression" class="size-4" />
        <mdi:text-box-outline v-else class="size-4" />
      </div>

      <div class="min-w-0 flex-1 space-y-3" :class="{ 'opacity-60': !alert.enabled }">
        <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
          <h4 class="font-semibold break-all">{{ alert.name }}</h4>
          <span v-if="!alert.enabled" class="status-pill status-pill-warning">
            {{ $t("notifications.alert.paused") }}
          </span>
          <mdi:arrow-right class="text-base-content/40 size-3.5" />
          <Popover class="group/dispatch" hover panel-class="bg-base-200 rounded-box w-48 p-2 shadow-lg">
            <template #trigger>
              <button
                type="button"
                class="hover:bg-base-300 flex items-center gap-1 rounded-md px-1.5 py-0.5 text-xs transition-colors"
                :class="alert.dispatcher ? 'text-base-content/70' : 'text-warning'"
              >
                <template v-if="alert.dispatcher">
                  <mdi:webhook v-if="alert.dispatcher.type === 'webhook'" class="size-3.5" />
                  <mdi:cloud v-else class="size-3.5" />
                  {{ alert.dispatcher.name }}
                </template>
                <template v-else>
                  <mdi:alert-outline class="size-3.5" />
                  {{ $t("notifications.alert.dispatcher-deleted") }}
                </template>
                <mdi:chevron-down class="size-3 opacity-0 transition-opacity group-hover/dispatch:opacity-100" />
              </button>
            </template>
            <ul class="menu w-full p-0">
              <li v-for="dest in dispatchers" :key="dest.id">
                <a
                  class="flex items-center gap-2"
                  :class="{ 'menu-active': dest.id === alert.dispatcher?.id }"
                  @click="changeDispatcher(dest.id)"
                >
                  <mdi:webhook v-if="dest.type === 'webhook'" />
                  <mdi:cloud v-else />
                  {{ dest.name }}
                </a>
              </li>
            </ul>
          </Popover>
        </div>

        <!-- Only the rule itself lives here: which containers, and what fires it. A fixed label
             column keeps the chips on the same line across every card, which an `auto` column
             could not do since each card sized it to its own longest label. The value column is
             `1fr` so long expressions wrap, but each chip is w-fit so a short one doesn't stretch
             a full-width bar across the card. -->
        <div class="text-base-content/60 grid grid-cols-[minmax(0,6rem)_1fr] items-baseline gap-x-3 gap-y-1.5 text-xs">
          <span>{{ $t("notifications.alert.containers") }}</span>
          <code
            class="bg-base-content/5 text-base-content/90 w-fit max-w-full rounded px-1.5 py-0.5 font-mono break-all"
            >{{ alert.containerExpression }}</code
          >
          <span>{{ ruleLabel }}</span>
          <code
            class="bg-base-content/5 text-base-content/90 w-fit max-w-full rounded px-1.5 py-0.5 font-mono break-all"
            >{{ ruleExpression }}</code
          >
        </div>
      </div>

      <input
        type="checkbox"
        class="toggle toggle-primary toggle-sm shrink-0"
        :checked="alert.enabled"
        @change="toggleEnabled"
      />
    </div>

    <!-- One line of what the rule did and how it is tuned. These are firing counts, not match
         counts: "0 containers" next to a filter that clearly matches something read as "this
         alert matches nothing". -->
    <div class="text-base-content/60 flex items-center justify-between gap-2 px-4 py-2 text-xs">
      <div class="flex min-w-0 flex-wrap items-center gap-x-4 gap-y-1">
        <span v-if="confirmingDelete" class="text-base-content">{{ $t("notifications.alert.delete-warning") }}</span>
        <template v-else>
          <span v-if="!alert.triggerCount && !firings.length">{{ $t("notifications.alert.never-triggered") }}</span>
          <template v-if="alert.triggerCount">
            <span>{{ $t("notifications.alert.triggered-count", alert.triggerCount) }}</span>
            <span>{{ $t("notifications.alert.containers-count", alert.triggeredContainers) }}</span>
          </template>
          <span v-if="lastTriggered">
            {{ $t("notifications.alert.last-triggered", { time: lastTriggered }) }}
          </span>
          <!--
            What Cloud remembers. The local counts above reset on restart; this is the half a
            local webhook cannot have, because nothing remembers a fire-and-forget POST.
          -->
          <span v-if="linked && firings.length" class="flex items-center gap-1">
            <mdi:history class="size-3.5 opacity-60" />
            <span class="text-base-content font-mono font-semibold">{{ firings.length }}</span>
            {{ $t("notifications.history.fired-recently") }}
          </span>
          <!-- Timing is how the rule is tuned, not what it matches, so it sits with the other
               metadata instead of competing with the expressions above. -->
          <span v-if="alert.metricExpression" class="text-base-content/40">
            {{
              $t("notifications.alert.window-meta", {
                duration: formatDuration(alert.sampleWindow || 15, locale || undefined),
              })
            }}
          </span>
          <span v-if="sendsImmediately" class="flex items-center gap-1" data-testid="alert-immediate">
            <mdi:lightning-bolt class="size-3.5 opacity-60" />
            {{ $t("notifications.alert-form.immediate-label") }}
          </span>
          <span v-if="cooldownSeconds" class="text-base-content/40">
            {{
              $t("notifications.alert.cooldown-meta", {
                duration: formatDuration(cooldownSeconds, locale || undefined),
              })
            }}
          </span>
        </template>
      </div>
      <!-- Deleting an alert is not undoable and there is no trash to restore it from. -->
      <div v-if="confirmingDelete" class="flex shrink-0 items-center gap-2">
        <button class="btn btn-xs" :disabled="isDeleting" @click="confirmingDelete = false">
          {{ $t("notifications.alert.delete-cancel") }}
        </button>
        <button class="btn btn-xs btn-error" :disabled="isDeleting" @click="deleteAlert">
          <span v-if="isDeleting" class="loading loading-spinner loading-xs"></span>
          {{ $t("notifications.alert.delete-confirm") }}
        </button>
      </div>
      <div v-else class="flex shrink-0 items-center">
        <button
          class="btn btn-ghost btn-sm btn-square"
          :aria-label="$t('notifications.destination.edit')"
          @click="editAlert"
        >
          <mdi:pencil-outline class="size-4" />
        </button>
        <button
          class="btn btn-ghost btn-sm btn-square"
          :aria-label="$t('notifications.alert.delete-confirm')"
          @click="confirmingDelete = true"
        >
          <mdi:trash-can-outline class="size-4" />
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { Dispatcher, NotificationRule } from "@/types/notifications";
import AlertForm from "./AlertForm.vue";

const { linked } = useCloudSurface();
const { forSubscription, fetchRecentAlerts } = useRecentAlerts();

const { alert, dispatchers, onUpdated, highlight } = defineProps<{
  alert: NotificationRule;
  dispatchers: Dispatcher[];
  onUpdated?: () => void;
  highlight?: boolean;
}>();

// Shared fetch: every card on the page reads the same rows.
onMounted(() => fetchRecentAlerts());
const firings = forSubscription(alert.id);

// The local timestamp resets on restart while Cloud's survives it, so the newer of the two.
const lastTriggered = computed(() => {
  void relativeTimeTick.value;
  const local = alert.lastTriggeredAt ? new Date(alert.lastTriggeredAt) : undefined;
  const times = [
    local && local.getFullYear() > 1 ? local.getTime() : 0,
    linked.value && firings.value.length ? firings.value[0].ts / 1e6 : 0,
  ];
  const newest = Math.max(...times);
  return newest ? toRelativeTime(new Date(newest), locale.value || undefined) : undefined;
});

const ruleLabel = computed(() => {
  if (alert.metricExpression) return t("notifications.alert.metric-filter");
  if (alert.eventExpression) return t("notifications.alert.event-filter");
  return t("notifications.alert.log-filter");
});
const ruleExpression = computed(() => alert.metricExpression || alert.eventExpression || alert.logExpression);

// Metric alerts default to 5m. Events and immediate log alerts only have one when set; other log alerts never do.
const cooldownSeconds = computed(() => {
  if (alert.metricExpression) return alert.cooldown || 300;
  if (alert.eventExpression || alert.immediate) return alert.cooldown || 0;
  return 0;
});

// A webhook has no Cloud review to skip, and moving a rule onto one leaves the flag saved.
// Deleting the destination leaves it too, on a rule that now sends nothing.
const sendsImmediately = computed(() => alert.immediate && !!alert.dispatcher && alert.dispatcher.type !== "webhook");

const isHighlighted = ref(highlight ?? false);
watch(
  () => highlight,
  (v) => {
    if (v) isHighlighted.value = true;
  },
);

const { t } = useI18n();
const { showToast } = useToast();
const showDrawer = useDrawer();
const isDeleting = ref(false);
const confirmingDelete = ref(false);

async function changeDispatcher(id: number) {
  await fetch(withBase(`/api/notifications/rules/${alert.id}`), {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ dispatcherId: id }),
  });
  onUpdated?.();
}

async function toggleEnabled() {
  await fetch(withBase(`/api/notifications/rules/${alert.id}`), {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ enabled: !alert.enabled }),
  });
  onUpdated?.();
}

function editAlert() {
  showDrawer(AlertForm, { alert, onCreated: onUpdated }, "lg");
}

async function deleteAlert() {
  isDeleting.value = true;
  try {
    const res = await fetch(withBase(`/api/notifications/rules/${alert.id}`), { method: "DELETE" });
    if (!res.ok) throw new Error((await res.json().catch(() => ({}))).error ?? res.statusText);
    confirmingDelete.value = false;
    onUpdated?.();
  } catch (e) {
    showToast({ type: "error", message: e instanceof Error ? escapeHtml(e.message) : t("error.something-went-wrong") });
  } finally {
    isDeleting.value = false;
  }
}
</script>

<style scoped>
.highlight-new {
  animation: highlight-fade 3s ease-out;
}
@keyframes highlight-fade {
  from {
    background-color: oklch(from var(--color-secondary) l c h / 0.25);
  }
  to {
    background-color: transparent;
  }
}
</style>
