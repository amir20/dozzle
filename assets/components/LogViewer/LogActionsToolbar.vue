<template>
  <div class="dropdown dropdown-end dropdown-hover">
    <label tabindex="0" class="btn btn-ghost btn-sm gap-0.5 px-2">
      <carbon:circle-solid class="w-2.5 text-red" v-if="streamConfig.stderr" />
      <carbon:circle-solid class="w-2.5 text-blue" v-if="streamConfig.stdout" />
    </label>
    <ul tabindex="0" class="menu dropdown-content rounded-box z-50 w-52 bg-base p-1 shadow">
      <li>
        <a @click.prevent="clear()">
          <octicon:trash-24 /> {{ $t("toolbar.clear") }}
          <key-shortcut char="k" :modifiers="['shift', 'meta']"></key-shortcut>
        </a>
      </li>
      <li>
        <a :href="`${base}/api/logs/download/${container.host}/${container.id}`" download>
          <octicon:download-24 /> {{ $t("toolbar.download") }}
        </a>
      </li>
      <li>
        <a @click.prevent="showSearch = true">
          <mdi:magnify /> {{ $t("toolbar.search") }}
          <key-shortcut char="f"></key-shortcut>
        </a>
      </li>
      <li class="line"></li>
      <li>
        <a
          @click="
            streamConfig.stdout = true;
            streamConfig.stderr = true;
          "
        >
          <div class="flex h-4 w-4 gap-0.5">
            <template v-if="streamConfig.stderr && streamConfig.stdout">
              <carbon:circle-solid class="w-2 text-red" />
              <carbon:circle-solid class="w-2 text-blue" />
            </template>
          </div>
          {{ $t("toolbar.show-all") }}
        </a>
      </li>
      <li>
        <a
          @click="
            streamConfig.stdout = true;
            streamConfig.stderr = false;
          "
        >
          <div class="flex h-4 w-4 flex-col gap-1">
            <carbon:circle-solid class="w-2 text-blue" v-if="!streamConfig.stderr && streamConfig.stdout" />
          </div>
          {{ $t("toolbar.show", { std: "STDOUT" }) }}
        </a>
      </li>
      <li>
        <a
          @click="
            streamConfig.stdout = false;
            streamConfig.stderr = true;
          "
        >
          <div class="flex h-4 w-4 flex-col gap-1">
            <carbon:circle-solid class="w-2 text-red" v-if="streamConfig.stderr && !streamConfig.stdout" />
          </div>
          {{ $t("toolbar.show", { std: "STDERR" }) }}
        </a>
      </li>
    </ul>
  </div>
</template>

<script lang="ts" setup>
const { showSearch } = useSearchFilter();
const { base } = config;

const clear = defineEmit();

const { container, streamConfig } = useContainerContext();
</script>

<style scoped lang="postcss">
li.line {
  @apply h-px bg-base-content/20;
}

a {
  @apply whitespace-nowrap;
}
</style>
