<template>
  <ScrollableView :scrollable="scrollable" v-if="host">
    <template #header>
      <div class="mx-2 flex items-center gap-2 md:ml-4">
        <div class="flex flex-1 gap-1.5 truncate @container md:gap-2">
          <div class="inline-flex font-mono text-sm">
            <div class="font-semibold">{{ host.name }}</div>
          </div>
        </div>
        <MultiContainerStat class="ml-auto" :containers="containers" />
        <MultiContainerActionToolbar class="mobile-hidden" @clear="viewer?.clear()" />
      </div>
    </template>
    <template #default>
      <ViewerWithSource
        ref="viewer"
        :stream-source="useHostStream"
        :entity="host"
        :visible-keys="new Map<string[], boolean>()"
      />
    </template>
  </ScrollableView>
</template>

<script lang="ts" setup>
import ViewerWithSource from "@/components/LogViewer/ViewerWithSource.vue";
import { ComponentExposed } from "vue-component-type-helpers";
const { id, scrollable = false } = defineProps<{
  id: string;
  scrollable?: boolean;
}>();
const store = useContainerStore();
const { containersByHost } = storeToRefs(store);
const { hosts } = useHosts();
const host = computed(() => hosts.value[id]);
const containers = computed(() => containersByHost.value[id] ?? []);
const viewer = useTemplateRef<ComponentExposed<typeof ViewerWithSource>>("viewer");
provideLoggingContext(containers, { showContainerName: true, showHostname: false });
</script>
