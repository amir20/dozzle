<template>
  <ScrollableView :scrollable="scrollable" v-if="containers.length && ready">
    <template #header>
      <div class="mx-2 flex items-center gap-1 md:ml-4 md:gap-2">
        <octicon:container-24 />
        <ContainerDropdown :containers="containers">{{ $t("label.container", containers.length) }}</ContainerDropdown>
        <MultiContainerStat class="ml-auto" :containers="containers" />
        <MultiContainerActionToolbar @clear="viewer?.clear()" />
      </div>
    </template>
    <template #default>
      <ViewerWithSource
        ref="viewer"
        :stream-source="useMergedStream"
        :entity="containers"
        :visible-keys="visibleKeys"
      />
    </template>
  </ScrollableView>
  <NotFound v-else-if="ready" :title="$t('error.container-not-found')" :hint="$t('error.container-not-found-hint')">
    <template #icon><octicon:container-24 class="size-5" /></template>
  </NotFound>
</template>

<script lang="ts" setup>
import ViewerWithSource from "@/components/logs/ViewerWithSource.vue";
import { ComponentExposed } from "vue-component-type-helpers";
import { Container } from "@/models/Container";

const { ids = [], scrollable = false } = defineProps<{
  ids?: string[];
  scrollable?: boolean;
}>();

const containerStore = useContainerStore();
const viewer = ref<ComponentExposed<typeof ViewerWithSource>>();
const { allContainersById, ready } = storeToRefs(containerStore);
// A container the store has dropped (removed, or replaced by a compose recreate)
// stays as the last object seen, so the view and its stream carry on and its stop
// still reads in the log. An id that never resolved, from a stale link, is left out.
// Plain map, not reactive: it only remembers, it never triggers.
const lastSeen = new Map<string, Container>();
const containers = computed(() =>
  ids.flatMap((id) => {
    const container = allContainersById.value[id] ?? lastSeen.get(id);
    if (!container) return [];
    lastSeen.set(id, container);
    return [container];
  }),
);
useMarkDropped(() => containers.value);

provideLoggingContext(containers, { showContainerName: true, showHostname: false });
const visibleKeys = useVisibleKeysByContainer();
</script>
