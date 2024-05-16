<template>
  <ScrollableView v-if="stack.name">
    <template #header>
      <div class="mx-2 flex items-center gap-2 md:ml-4">
        <div class="flex flex-1 gap-1.5 truncate @container md:gap-2">
          <div class="inline-flex font-mono text-sm">
            <div class="font-semibold">{{ stack.name }}</div>
          </div>
        </div>
      </div>
    </template>
    <template #default="{ setLoading }">
      <StackViewerWithSource ref="viewer" @loading-more="setLoading($event)" />
    </template>
  </ScrollableView>
</template>

<script lang="ts" setup>
import { Stack } from "@/models/Stack";
import StackViewerWithSource from "./StackViewerWithSource.vue";

const { name } = defineProps<{
  name: string;
}>();

const store = useStackStore();
const { stacks } = storeToRefs(store);
const stack = computed(() => stacks.value.find((s) => s.name === name) ?? new Stack("", []));

provideStackContext(stack);

const viewer = ref<InstanceType<typeof StackViewerWithSource>>();

const onClearClicked = () => viewer.value?.clear();

onKeyStroke("k", (e) => {
  if ((e.ctrlKey || e.metaKey) && e.shiftKey) {
    onClearClicked();
    e.preventDefault();
  }
});
</script>
