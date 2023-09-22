<template>
  <search></search>
  <log-container
    :id="id"
    :show-title="true"
    :scrollable="activeContainers.length > 0"
    v-if="currentContainer"
  ></log-container>
  <div v-else-if="ready" class="hero min-h-screen bg-base-200">
    <div class="hero-content text-center">
      <div class="max-w-md">
        <p class="py-6 text-2xl font-bold">{{ $t("error.container-not-found") }}</p>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import search from "@/components/Search.vue";
const store = useContainerStore();
const { id } = defineProps<{ id: string }>();

const currentContainer = store.currentContainer($$(id));
const { activeContainers, ready } = storeToRefs(store);

watchEffect(() => {
  if (ready.value) {
    if (currentContainer.value) {
      setTitle(currentContainer.value.name);
    } else {
      setTitle("Not Found");
    }
  }
});
</script>
