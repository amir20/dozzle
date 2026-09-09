<template>
  <!--
    Every meter the plan actually caps, in one block, with the billing period stated
    once underneath rather than repeated under each bar.

    The two newer meters are guarded: an instance can be pointed at a cloud that has
    not shipped them yet, and a meter reading 0 / 0 would claim the account had no
    agent chats left rather than admitting the number is unknown.
  -->
  <div class="flex flex-col" :class="compact ? 'gap-3' : 'gap-4'">
    <UsageMeter :label="$t('cloud.usage')" :used="usage.events_used" :limit="usage.events_limit" :compact="compact" />
    <UsageMeter
      v-if="usage.agent_messages_limit"
      :label="$t('cloud.usage-agent-messages')"
      :used="usage.agent_messages_used ?? 0"
      :limit="usage.agent_messages_limit"
      :compact="compact"
    />
    <UsageMeter
      v-if="usage.log_bytes_limit"
      :label="$t('cloud.usage-log-bytes')"
      :used="usage.log_bytes_used ?? 0"
      :limit="usage.log_bytes_limit"
      unit="bytes"
      :compact="compact"
    />
    <div v-if="usage.period" class="text-base-content/40 font-mono" :class="compact ? 'text-[0.6875rem]' : 'text-xs'">
      {{ usage.period }}
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { CloudStatus } from "@/types/notifications";

defineProps<{ usage: CloudStatus["usage"]; compact?: boolean }>();
</script>
