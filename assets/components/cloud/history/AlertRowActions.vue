<template>
  <!-- The button Cloud cannot have: the real stream at the moment it matched. -->
  <button type="button" class="btn btn-ghost btn-xs" :title="label" @click="showLines">
    <material-symbols:eye-tracking class="size-4" />
    <span class="hidden sm:inline">{{ label }}</span>
  </button>
</template>

<script lang="ts" setup>
import type { LogMoment } from "@/composable/logs/logJump";

const { moment } = defineProps<{ moment: LogMoment }>();
const { jumpTo } = useLogJump();
const { closeRail } = useCloudRail();

// From the cloud drawer the stream is under a modal, so the drawer goes first.
function showLines() {
  closeRail();
  jumpTo(moment);
}
const { t } = useI18n();

// A CPU spike or a container event has no line to show — logId is absent for
// exactly those — so promising lines sends the reader looking for something
// that was never there. What they get either way is the stream around the
// moment; only the log-anchored case can point at the line itself.
const label = computed(() =>
  moment.logId ? t("notifications.history.show-lines") : t("notifications.history.show-moment"),
);
</script>
