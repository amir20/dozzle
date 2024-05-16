<template>
  <ScrollableView>
    <template #header>
      <div class="mx-2 flex items-center gap-2 md:ml-4">
        {{ name }}
      </div>
    </template>
    <template #default="{ setLoading }">
      <StackViewerWithSource ref="viewer" @loading-more="setLoading($event)" />
    </template>
  </ScrollableView>
</template>

<script lang="ts" setup>
import StackViewerWithSource from "./StackViewerWithSource.vue";

const { name } = defineProps<{
  name: string;
}>();

const store = useStackStore();
const { stacks } = storeToRefs(store);
const stack = computed(() => stacks.value.find((s) => s.name === name));

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
