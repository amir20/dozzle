<template>
  <!-- The button Cloud cannot have. Cloud links out to its own page; here the
       same alert opens the real stream at the moment it fired. -->
  <button
    type="button"
    class="btn btn-ghost btn-xs"
    :title="label"
    @click="jumpTo({ containerId: alert.containerId, date: new Date(alert.ts / 1e6), logId: alert.logId })"
  >
    <material-symbols:eye-tracking class="size-4" />
    <span class="hidden sm:inline">{{ label }}</span>
  </button>
  <a
    v-if="alert.url"
    :href="alert.url"
    target="_blank"
    rel="noreferrer noopener"
    class="btn btn-ghost btn-xs btn-square"
    :title="$t('notifications.history.open-in-cloud')"
  >
    <mdi:open-in-new class="size-3.5 opacity-40" />
  </a>
</template>

<script lang="ts" setup>
import type { CloudAlert } from "@/composable/cloud/cloudAlerts";

const { alert } = defineProps<{ alert: CloudAlert }>();
const { jumpTo } = useLogJump();
const { t } = useI18n();

// A CPU spike or a container event has no line to show — logId is absent for
// exactly those — so promising lines sends the reader looking for something
// that was never there. What they get either way is the stream around the
// moment; only the log-anchored case can point at the line itself.
const label = computed(() =>
  alert.logId ? t("notifications.history.show-lines") : t("notifications.history.show-moment"),
);
</script>
