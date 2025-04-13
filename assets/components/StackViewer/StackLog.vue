<template>
  <ScrollableView :scrollable="scrollable" v-if="stack.name">
    <template #header>
      <div class="mx-2 flex items-center gap-2 md:ml-4">
        <div class="@container flex flex-1 items-center gap-1.5 truncate md:gap-2">
          <ph:stack />
          <div class="inline-flex font-mono text-sm">
            <div class="font-semibold">{{ stack.name }}</div>
          </div>
          <Tag class="hidden font-mono max-md:hidden @3xl:block" size="small">
            {{ $t("label.container", stack.containers.length) }}
          </Tag>
          <Tag class="hidden font-mono max-md:hidden @3xl:block" size="small">
            {{ $t("label.service", stack.services.length) }}
          </Tag>
        </div>
        <MultiContainerStat class="ml-auto" :containers="stack.containers" />
        <MultiContainerActionToolbar class="max-md:hidden" @clear="viewer?.clear()" />
      </div>
    </template>
    <template #default>
      <ViewerWithSource
        ref="viewer"
        :stream-source="useStackStream"
        :entity="stack"
        :visible-keys="new Map<string[], boolean>()"
      />
    </template>
  </ScrollableView>
</template>

<script lang="ts" setup>
import { Stack } from "@/models/Stack";
import ViewerWithSource from "@/components/LogViewer/ViewerWithSource.vue";
import { ComponentExposed } from "vue-component-type-helpers";
const { name, scrollable = false } = defineProps<{
  scrollable?: boolean;
  name: string;
}>();

const viewer = ref<ComponentExposed<typeof ViewerWithSource>>();
const store = useSwarmStore();
const { stacks } = storeToRefs(store) as unknown as { stacks: Ref<Stack[]> };
const stack = computed(() => stacks.value.find((s) => s.name === name) ?? new Stack("", [], []));
provideLoggingContext(
  toRef(() => stack.value.containers),
  { showContainerName: true, showHostname: false },
);
</script>
