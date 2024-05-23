<template>
  <ScrollableView :scrollable="scrollable" v-if="group.containers.length && ready">
    <template #header>
      <div class="mx-2 flex items-center gap-2 md:ml-4">
        <div class="flex flex-1 gap-1.5 truncate @container md:gap-2">
          <div class="inline-flex font-mono text-sm">
            <div class="font-semibold">{{ group.containers.length }} containers</div>
          </div>
        </div>
        <MultiContainerStat class="ml-auto" :containers="group.containers" />
      </div>
    </template>
    <template #default="{ setLoading }">
      <ViewerWithSource
        ref="viewer"
        @loading-more="setLoading($event)"
        :stream-source="useGroupedStream"
        :entity="group"
        :visible-keys="visibleKeys"
        :show-container-name="true"
      />
    </template>
  </ScrollableView>
</template>

<script lang="ts" setup>
import ViewerWithSource from "@/components/LogViewer/ViewerWithSource.vue";
import { GroupedContainers } from "@/models/Container";

const { name, scrollable = false } = defineProps<{
  name: string;
  scrollable?: boolean;
}>();

const containerStore = useContainerStore();

const { ready } = storeToRefs(containerStore);

const swarmStore = useSwarmStore();
const { customGroups } = storeToRefs(swarmStore);

const group = computed(() => customGroups.value.find((g) => g.name === name) ?? new GroupedContainers("", []));

const visibleKeys = ref<string[][]>([]);
provideLoggingContext(toRef(() => group.value.containers));
</script>
