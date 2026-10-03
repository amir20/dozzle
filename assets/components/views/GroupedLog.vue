<template>
  <ScrollableView :scrollable="scrollable" v-if="found && ready">
    <template #header>
      <div class="mx-2 flex items-center gap-2 md:ml-4">
        <ContainerDropdown :containers="group.containers">
          {{ $t("label.container", group.containers.length) }}
        </ContainerDropdown>
        <MultiContainerStat class="ml-auto" :containers="group.containers" />
        <MultiContainerActionToolbar class="max-md:hidden" :name="group.name" @clear="viewer?.clear()" />
      </div>
    </template>
    <template #default>
      <ViewerWithSource ref="viewer" :stream-source="useGroupedStream" :entity="group" :visible-keys="visibleKeys" />
    </template>
  </ScrollableView>
  <NotFound v-else-if="ready" :title="$t('error.nothing-running')" :hint="$t('error.nothing-running-hint')">
    <template #icon><octicon:container-24 class="size-5" /></template>
  </NotFound>
</template>

<script lang="ts" setup>
import ViewerWithSource from "@/components/logs/ViewerWithSource.vue";
import { GroupedContainers } from "@/models/Container";
import { ComponentExposed } from "vue-component-type-helpers";

const { name, scrollable = false } = defineProps<{
  name: string;
  scrollable?: boolean;
}>();

const containerStore = useContainerStore();
const viewer = ref<ComponentExposed<typeof ViewerWithSource>>();

const { ready } = storeToRefs(containerStore);

const swarmStore = useSwarmStore();
const { customGroups } = storeToRefs(swarmStore);

// Held while its members stop and restart, so the view keeps its stream and scrollback.
const found = useStickyEntity(
  () => customGroups.value.find((g) => g.name === name),
  () => name,
);
const group = computed(() => found.value ?? new GroupedContainers("", []));

provideLoggingContext(
  toRef(() => group.value.containers),
  { showContainerName: true, showHostname: false },
);
const visibleKeys = useVisibleKeysByContainer();
</script>
