<template>
  <div class="card bg-base-100 shadow-sm" :class="{ 'opacity-60': !alert.enabled }">
    <div class="card-body gap-4 p-5">
      <!-- Header -->
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-2">
          <h4 class="flex items-center gap-2 text-lg font-semibold">
            <mdi:chart-line v-if="alert.metricExpression" class="text-info" />
            <mdi:text-box-outline v-else class="text-info" />
            <span>{{ alert.name }}</span> <span class="text-sm font-light">→</span>
            <span class="flex gap-1 text-xs font-light" :class="{ 'text-warning': !alert.dispatcher }">
              <template v-if="alert.dispatcher">
                <mdi:webhook v-if="alert.dispatcher.type === 'webhook'" />
                <mdi:cloud v-else />
                {{ alert.dispatcher.name }}
              </template>
              <template v-else>
                <mdi:alert-outline />
                {{ $t("notifications.alert.dispatcher-deleted") }}
              </template>
            </span>
          </h4>
          <span v-if="!alert.enabled" class="badge badge-warning badge-sm">{{ $t("notifications.alert.paused") }}</span>
        </div>
        <input type="checkbox" class="toggle toggle-primary" :checked="alert.enabled" @change="toggleEnabled" />
      </div>

      <!-- Expressions -->
      <div class="text-base-content/80 grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
        <span>{{ $t("notifications.alert.containers") }}</span>
        <code class="bg-base-200 text-base-content rounded px-2 py-0.5 font-mono">{{ alert.containerExpression }}</code>
        <template v-if="alert.metricExpression">
          <span>{{ $t("notifications.alert.metric-filter") }}</span>
          <code class="bg-base-200 text-base-content rounded px-2 py-0.5 font-mono">{{ alert.metricExpression }}</code>
          <span>{{ $t("notifications.alert.cooldown") }}</span>
          <span>{{ alert.cooldown || 300 }}s</span>
        </template>
        <template v-else>
          <span>{{ $t("notifications.alert.log-filter") }}</span>
          <code class="bg-base-200 text-base-content rounded px-2 py-0.5 font-mono">{{ alert.logExpression }}</code>
        </template>
      </div>

      <!-- Footer -->
      <div class="border-base-content/10 text-base-content/80 flex items-center justify-between border-t pt-3 text-xs">
        <div class="flex items-center gap-4">
          <span>
            {{ $t("notifications.alert.containers-count", { count: alert.triggeredContainers }) }}
          </span>
          <span>
            {{ $t("notifications.alert.triggered-count", { count: alert.triggerCount }) }}
          </span>
          <span v-if="alert.lastTriggeredAt">
            {{ $t("notifications.alert.last-triggered", { time: formatTimeAgo(alert.lastTriggeredAt) }) }}
          </span>
        </div>
        <div class="flex items-center gap-1">
          <button class="btn btn-ghost btn-square" @click="editAlert">
            <mdi:pencil-outline />
          </button>
          <button class="btn btn-ghost btn-square" @click="deleteAlert" :disabled="isDeleting">
            <span v-if="isDeleting" class="loading loading-spinner loading-xs"></span>
            <mdi:trash-can-outline v-else />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { NotificationRule } from "@/types/notifications";
import AlertForm from "./AlertForm.vue";

const { alert, onUpdated } = defineProps<{
  alert: NotificationRule;
  onUpdated?: () => void;
}>();

const showDrawer = useDrawer();
const isDeleting = ref(false);

function formatTimeAgo(dateStr: string): string {
  const date = new Date(dateStr);
  if (date.getFullYear() === 0) return "-";
  return toRelativeTime(date, undefined);
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
    await fetch(withBase(`/api/notifications/rules/${alert.id}`), { method: "DELETE" });
    onUpdated?.();
  } finally {
    isDeleting.value = false;
  }
}
</script>
