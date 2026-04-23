<template>
  <ScrollableView :scrollable="scrollable" v-if="groupHosts.length > 0">
    <template #header>
      <div class="mx-2 flex items-center gap-2 md:ml-4">
        <div class="flex flex-1 items-center gap-1.5 truncate md:gap-2">
          <ph:computer-tower />
          <div class="inline-flex font-mono text-sm">
            <div class="font-semibold">{{ name }}</div>
          </div>
          <Tag class="font-mono max-md:hidden" size="small">
            {{ $t("label.host-count", groupHosts.length) }}
          </Tag>
          <Tag class="font-mono max-md:hidden" size="small">
            {{ $t("label.container", containers.length) }}
          </Tag>
        </div>
        <MultiContainerStat class="ml-auto" :containers="containers" />
        <MultiContainerActionToolbar class="max-md:hidden" :name="name" @clear="viewer?.clear()" />
      </div>
    </template>
    <template #default>
      <ViewerWithSource
        ref="viewer"
        :stream-source="useHostGroupStream"
        :entity="groupRef"
        :visible-keys="new Map<string[], boolean>()"
      />
    </template>
  </ScrollableView>
</template>

<script lang="ts" setup>
import ViewerWithSource from "@/components/LogViewer/ViewerWithSource.vue";
import { useHostGroupStream } from "@/composable/eventStreams";
import { ComponentExposed } from "vue-component-type-helpers";

const { name, scrollable = false } = defineProps<{
  name: string;
  scrollable?: boolean;
}>();

const { hosts } = useHosts();
const store = useContainerStore();
const { containersByHost } = storeToRefs(store);

const groupHosts = computed(() => Object.values(hosts.value).filter((h) => h.group === name));
const groupRef = computed(() => ({ name }));

const containers = computed(() =>
  groupHosts.value.flatMap((h) => containersByHost.value?.[h.id]?.filter((c) => c.state === "running") ?? []),
);

const viewer = useTemplateRef<ComponentExposed<typeof ViewerWithSource>>("viewer");
provideLoggingContext(containers, { showContainerName: true, showHostname: true });
</script>
