<template>
  <!-- The header's way into Cloud before it is linked. Once linked, the rail's
       cloud mark takes over and this is not mounted. -->
  <Dropdown :placement="placement" :trigger-class="triggerClass" data-testid="cloud-link">
    <template #trigger>
      <mdi:cloud class="icon-float text-base-content/40" :class="iconClass" />
      <span class="sr-only">{{ $t("cloud.title") }}</span>
    </template>
    <template #content>
      <div class="w-76">
        <div class="flex items-center gap-2.5 px-2 py-1.5">
          <div class="bg-info/10 text-info shrink-0 rounded-full p-1.5">
            <mdi:cloud class="size-5" />
          </div>
          <span class="text-sm font-semibold">{{ $t("cloud.title") }}</span>
        </div>

        <div class="bg-base-content/10 my-1.5 h-px"></div>

        <ul class="text-base-content/70 space-y-2 px-2 py-1 text-xs">
          <CloudPitch />
        </ul>

        <div class="bg-base-content/10 my-1.5 h-px"></div>

        <div class="flex gap-2 px-1 pb-0.5">
          <a :href="config.cloudUrl" target="_blank" rel="noreferrer noopener" class="btn btn-sm flex-1">
            {{ $t("cloud.learn-more") }}
          </a>
          <button class="btn btn-primary btn-sm flex-1" @click="startCloudLink('cloud')">
            <mdi:link-variant class="text-base" />
            {{ $t("cloud.link-instance") }}
          </button>
        </div>
      </div>
    </template>
  </Dropdown>
</template>

<script lang="ts" setup>
const {
  placement = "bottom-end",
  iconClass = "size-6",
  triggerClass,
} = defineProps<{
  placement?: PopoverPlacement;
  iconClass?: string;
  triggerClass?: string;
}>();
</script>
