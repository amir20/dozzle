<template>
  <!--
    Every meter the plan actually caps, in one block, with the billing period stated
    once underneath rather than repeated under each bar.

    The two newer meters are guarded: an instance can be pointed at a cloud that has
    not shipped them yet, and a meter reading 0 / 0 would claim the account had no
    agent chats left rather than admitting the number is unknown.

    The `row` variant lays the same meters side by side and drops the percentage line
    under each bar. On a settings pane that runs the width of the page, three stacked
    meters cost a screenful for three numbers that fit on one line.
  -->
  <div :class="row ? 'flex flex-col gap-2' : compact ? 'flex flex-col gap-3' : 'flex flex-col gap-4'">
    <div
      :class="row ? 'grid gap-x-8 gap-y-3 @xl:grid-cols-3' : compact ? 'flex flex-col gap-3' : 'flex flex-col gap-4'"
    >
      <UsageMeter
        :label="$t('cloud.usage')"
        :used="usage.events_used"
        :limit="usage.events_limit"
        :compact="compact"
        :hide-percent="row"
      />
      <UsageMeter
        v-if="usage.agent_messages_limit"
        :label="$t('cloud.usage-agent-messages')"
        :used="usage.agent_messages_used ?? 0"
        :limit="usage.agent_messages_limit"
        :compact="compact"
        :hide-percent="row"
      />
      <UsageMeter
        v-if="usage.log_bytes_limit"
        :label="$t('cloud.usage-log-bytes')"
        :used="usage.log_bytes_used ?? 0"
        :limit="usage.log_bytes_limit"
        unit="bytes"
        :compact="compact"
        :hide-percent="row"
      />
    </div>
    <div
      v-if="usage.period"
      class="text-base-content/40 font-mono"
      :class="[compact ? 'text-[0.6875rem]' : 'text-xs', row ? 'text-right' : '']"
    >
      {{ usage.period }}
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { CloudStatus } from "@/types/notifications";

defineProps<{ usage: CloudStatus["usage"]; compact?: boolean; row?: boolean }>();
</script>
