<template>
  <ScrollableView :scrollable="scrollable" v-if="service.name">
    <template #header>
      <div class="mx-2 flex items-center gap-2 md:ml-4">
        <div class="flex flex-1 gap-1.5 truncate @container md:gap-2">
          <div class="inline-flex font-mono text-sm">
            <div class="font-semibold">{{ service.name }}</div>
          </div>
        </div>
      </div>
    </template>
    <template #default="{ setLoading }">
      <ViewerWithSource
        ref="viewer"
        @loading-more="setLoading($event)"
        :stream-source="useServiceContextLogStream"
        :visible-keys="visibleKeys"
        :show-container-name="true"
      />
    </template>
  </ScrollableView>
</template>

<script lang="ts" setup>
import { Service } from "@/models/Stack";
import ViewerWithSource from "@/components/LogViewer/ViewerWithSource.vue";

const { name, scrollable = false } = defineProps<{
  scrollable?: boolean;
  name: string;
}>();

const visibleKeys = ref<string[][]>([]);

const store = useSwarmStore();
const { services } = storeToRefs(store) as unknown as { services: Ref<Service[]> };
const service = computed(() => services.value.find((s) => s.name === name) ?? new Service("", []));

provideServiceContext(service);

const viewer = ref<InstanceType<typeof ViewerWithSource>>();

const onClearClicked = () => viewer.value?.clear();

onKeyStroke("k", (e) => {
  if ((e.ctrlKey || e.metaKey) && e.shiftKey) {
    onClearClicked();
    e.preventDefault();
  }
});
</script>
