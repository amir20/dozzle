<template>
  <ScrollableView :scrollable="scrollable" v-if="containers.length && ready">
    <template #header>
      <div class="mx-2 flex items-center gap-2 md:ml-4">
        <div class="@container flex flex-1 items-center gap-1.5 truncate md:gap-2">
          <octicon:container-24 />
          <div class="inline-flex font-mono text-sm">
            <div class="font-semibold">{{ containers.length }} containers</div>
          </div>
        </div>
        <MultiContainerStat class="ml-auto" :containers="containers" />
        <MultiContainerActionToolbar class="max-md:hidden" @clear="viewer?.clear()" />
      </div>
    </template>
    <template #default>
      <ViewerWithSource
        ref="viewer"
        :stream-source="useMergedStream"
        :entity="containers"
        :visible-keys="new Map<string[], boolean>()"
      />
    </template>
  </ScrollableView>
</template>

<script lang="ts" setup>
import ViewerWithSource from "@/components/LogViewer/ViewerWithSource.vue";
import { ComponentExposed } from "vue-component-type-helpers";

const { ids = [], scrollable = false } = defineProps<{
  ids?: string[];
  scrollable?: boolean;
}>();

const containerStore = useContainerStore();
const viewer = ref<ComponentExposed<typeof ViewerWithSource>>();
const { allContainersById, ready } = storeToRefs(containerStore);
const containers = computed(() => ids.map((id) => allContainersById.value[id]));

provideLoggingContext(containers, { showContainerName: true, showHostname: false });
</script>
