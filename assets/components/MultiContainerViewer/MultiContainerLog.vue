<template>
  <ScrollableView :scrollable="scrollable" v-if="containers.length && ready">
    <template #header>
      <div class="mx-2 flex items-center gap-2 md:ml-4">
        <div class="flex flex-1 gap-1.5 truncate @container md:gap-2">
          <div class="inline-flex font-mono text-sm">
            <div class="font-semibold">{{ containers.length }} containers</div>
          </div>
        </div>
        <MultiContainerStat class="ml-auto" :containers="containers" />
      </div>
    </template>
    <template #default="{ setLoading }">
      <ViewerWithSource
        ref="viewer"
        @loading-more="setLoading($event)"
        :stream-source="useMergedStream"
        :entity="containers"
        :visible-keys="visibleKeys"
        :show-container-name="true"
      />
    </template>
  </ScrollableView>
</template>

<script lang="ts" setup>
import ViewerWithSource from "@/components/LogViewer/ViewerWithSource.vue";

const { ids = [], scrollable = false } = defineProps<{
  ids?: string[];
  scrollable?: boolean;
}>();

const containerStore = useContainerStore();

const { allContainersById, ready } = storeToRefs(containerStore);

const containers = computed(() => ids.map((id) => allContainersById.value[id]));

const visibleKeys = ref<string[][]>([]);

provideLoggingContext(containers);
</script>
