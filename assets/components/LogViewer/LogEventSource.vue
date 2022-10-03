<template>
  <infinite-loader :onLoadMore="fetchMore" :enabled="messages.length > 100"></infinite-loader>
  <slot :messages="messages"></slot>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import { type ComputedRef } from "vue";

const emit = defineEmits<{
  (e: "loading-more", value: boolean): void;
}>();

const container = inject("container") as ComputedRef<Container>;
const { messages, loadOlderLogs } = useLogStream(container);

const beforeLoading = () => emit("loading-more", true);
const afterLoading = () => emit("loading-more", false);

defineExpose({
  clear: () => (messages.value = []),
});

const fetchMore = () => loadOlderLogs({ beforeLoading, afterLoading });
</script>
