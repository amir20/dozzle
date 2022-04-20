<template>
  <search></search>
  <log-container :id="id" show-title :scrollable="activeContainers.length > 0"> </log-container>
</template>

<script lang="ts" setup>
import { onMounted, toRefs, watchEffect } from "vue";
import Search from "@/components/Search.vue";
import LogContainer from "@/components/LogContainer.vue";
import { setTitle } from "@/composables/title";
import { useContainerStore } from "@/stores/container";
import { storeToRefs } from "pinia";

const store = useContainerStore();

const props = defineProps({
  id: {
    type: String,
    required: true,
  },
});

const { id } = toRefs(props);

const currentContainer = store.currentContainer(id);
const { activeContainers } = storeToRefs(store);

setTitle("loading");

onMounted(() => {
  setTitle(currentContainer.value?.name);
});

watchEffect(() => setTitle(currentContainer.value?.name));
</script>
