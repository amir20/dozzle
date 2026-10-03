<template>
  <ScrollableView :scrollable="scrollable" v-if="found">
    <template #header>
      <div class="mx-2 flex items-center gap-2 md:ml-4">
        <ph:stack-simple />
        <ContainerDropdown :containers="service.containers">{{ service.name }}</ContainerDropdown>
        <MultiContainerStat class="ml-auto" :containers="service.containers" />
        <MultiContainerActionToolbar class="max-md:hidden" :name="service.name" @clear="viewer?.clear()" />
      </div>
    </template>
    <template #default>
      <ViewerWithSource ref="viewer" :stream-source="useServiceStream" :entity="service" :visible-keys="visibleKeys" />
    </template>
  </ScrollableView>
  <NotFound v-else-if="ready" :title="$t('error.nothing-running')" :hint="$t('error.nothing-running-hint')">
    <template #icon><ph:stack-simple class="size-5" /></template>
  </NotFound>
</template>

<script lang="ts" setup>
import { Service } from "@/models/Stack";
import ViewerWithSource from "@/components/logs/ViewerWithSource.vue";
import { ComponentExposed } from "vue-component-type-helpers";

const { name, scrollable = false } = defineProps<{
  scrollable?: boolean;
  name: string;
}>();

const viewer = ref<ComponentExposed<typeof ViewerWithSource>>();
const store = useSwarmStore();
const { services } = storeToRefs(store) as unknown as { services: Ref<Service[]> };
// Held while its replicas stop and restart, so the view keeps its stream and scrollback.
const found = useStickyEntity(
  () => services.value.find((s) => s.name === name),
  () => name,
);
const service = computed(() => found.value ?? new Service("", []));
const { ready } = storeToRefs(useContainerStore());
const visibleKeys = useVisibleKeysByContainer();

provideLoggingContext(
  toRef(() => service.value.containers),
  { showContainerName: true, showHostname: false },
);
</script>
