<template>
  <ScrollableView :scrollable="scrollable" v-if="owner.name">
    <template #header>
      <div class="mx-2 flex items-center gap-2 md:ml-4">
        <div class="@container flex flex-1 items-center gap-1.5 md:gap-2">
          <ph:stack-simple />
          <div class="font-mono text-sm font-semibold">{{ owner.kind }}/{{ owner.name }}</div>
          <ContainerDropdown :containers="owner.containers">
            {{ $t("label.container", owner.containers.length) }}
          </ContainerDropdown>
        </div>
        <MultiContainerStat class="ml-auto" :containers="owner.containers" />
        <MultiContainerActionToolbar class="max-md:hidden" @clear="viewer?.clear()" />
      </div>
    </template>
    <template #default>
      <ViewerWithSource
        ref="viewer"
        :stream-source="useOwnerStream"
        :entity="owner"
        :visible-keys="new Map<string[], boolean>()"
      />
    </template>
  </ScrollableView>
</template>

<script lang="ts" setup>
import { K8sOwner } from "@/stores/k8s";
import ViewerWithSource from "@/components/LogViewer/ViewerWithSource.vue";
import { ComponentExposed } from "vue-component-type-helpers";

const { owner, scrollable = false } = defineProps<{
  scrollable?: boolean;
  owner: K8sOwner;
}>();

const viewer = ref<ComponentExposed<typeof ViewerWithSource>>();

provideLoggingContext(
  toRef(() => owner.containers),
  { showContainerName: true, showHostname: false },
);
</script>
