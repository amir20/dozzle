<template>
  <div class="flex min-w-0 items-center gap-2 text-xs leading-5">
    <span v-if="step.state === 'running'" class="loading loading-spinner text-info size-3 shrink-0"></span>
    <mdi:alert-outline v-else-if="step.state === 'failed'" class="text-warning size-3.5 shrink-0" />
    <component :is="stepIcon(step.tool)" v-else class="text-base-content/40 size-3.5 shrink-0" />
    <span class="text-base-content/70 shrink-0">{{ stepText(step) }}</span>
    <span v-if="step.detail" class="text-base-content/40 min-w-0 truncate font-mono">{{ step.detail }}</span>
    <span v-if="result" class="text-base-content/40 ml-auto shrink-0 pl-3 tabular-nums">{{ result }}</span>
  </div>
</template>

<script lang="ts" setup>
import type { ChatStep } from "@/composable/cloud/cloudChat";
import { stepIcon, stepResult, stepText } from "./chatSteps";

// One lookup in the trail: what it did, what it was about, what came back.
const props = defineProps<{ step: ChatStep }>();
const result = computed(() => stepResult(props.step));
</script>
