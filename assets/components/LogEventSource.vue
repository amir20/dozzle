<template>
  <infinite-loader :onLoadMore="fetchMore" :enabled="messages.length > 100"></infinite-loader>
  <slot :messages="messages"></slot>
</template>

<script lang="ts" setup>
import { useEventSource } from "@/composables/eventsource";
import { Container } from "@/types/Container";
import { inject, ComputedRef } from "vue";

const emit = defineEmits(["loading-more"]);
const container = inject("container") as ComputedRef<Container>;
const { connect, messages, loadOlderLogs } = useEventSource(container);

const beforeLoading = () => emit("loading-more", true);
const afterLoading = () => emit("loading-more", false);

defineExpose({
  clear: () => (messages.value = []),
});

const fetchMore = () => loadOlderLogs({ beforeLoading, afterLoading });

connect();
</script>
