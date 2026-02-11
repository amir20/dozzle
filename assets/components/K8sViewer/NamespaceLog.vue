<template>
  <ScrollableView :scrollable="scrollable" v-if="namespace.name">
    <template #header>
      <div class="mx-2 flex items-center gap-2 md:ml-4">
        <div class="@container flex flex-1 items-center gap-1.5 md:gap-2">
          <ph:stack />
          <div class="font-mono text-sm font-semibold">{{ namespace.name }}</div>
          <ContainerDropdown :containers="namespace.containers">
            {{ $t("label.container", namespace.containers.length) }}
          </ContainerDropdown>
        </div>
        <MultiContainerStat class="ml-auto" :containers="namespace.containers" />
        <MultiContainerActionToolbar class="max-md:hidden" :name="namespace.name" @clear="viewer?.clear()" />
      </div>
    </template>
    <template #default>
      <ViewerWithSource
        ref="viewer"
        :stream-source="useNamespaceStream"
        :entity="namespace"
        :visible-keys="new Map<string[], boolean>()"
      />
    </template>
  </ScrollableView>
</template>

<script lang="ts" setup>
import { K8sNamespace } from "@/stores/k8s";
import ViewerWithSource from "@/components/LogViewer/ViewerWithSource.vue";
import { ComponentExposed } from "vue-component-type-helpers";

const { namespace, scrollable = false } = defineProps<{
  scrollable?: boolean;
  namespace: K8sNamespace;
}>();

const viewer = ref<ComponentExposed<typeof ViewerWithSource>>();

provideLoggingContext(
  toRef(() => namespace.containers),
  { showContainerName: true, showHostname: false },
);
</script>
