<template>
  <div class="dropdown dropdown-end dropdown-hover">
    <label tabindex="0" class="btn btn-ghost btn-sm gap-0.5 px-2">
      <carbon:circle-solid class="text-red w-2.5" v-if="streamConfig.stderr" />
      <carbon:circle-solid class="text-blue w-2.5" v-if="streamConfig.stdout" />
    </label>
    <ul
      tabindex="0"
      class="menu dropdown-content rounded-box bg-base-200 border-base-content/20 z-50 w-52 border p-1 shadow-sm"
      @click="hideMenu"
    >
      <li>
        <a @click.prevent="clear()">
          <octicon:trash-24 /> {{ $t("toolbar.clear") }}
          <KeyShortcut char="k" :modifiers="['shift', 'meta']" />
        </a>
      </li>
      <li>
        <a :href="downloadUrl" download> <octicon:download-24 /> {{ $t("toolbar.download") }} </a>
      </li>
      <li>
        <a @click.prevent="showSearch = true">
          <mdi:magnify /> {{ $t("toolbar.search") }}
          <KeyShortcut char="f" />
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
          <div class="flex size-4 gap-0.5">
            <template v-if="streamConfig.stderr && streamConfig.stdout">
              <carbon:circle-solid class="text-red w-2" />
              <carbon:circle-solid class="text-blue w-2" />
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
          <div class="flex size-4 flex-col gap-1">
            <carbon:circle-solid class="text-blue w-2" v-if="!streamConfig.stderr && streamConfig.stdout" />
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
          <div class="flex size-4 flex-col gap-1">
            <carbon:circle-solid class="text-red w-2" v-if="streamConfig.stderr && !streamConfig.stdout" />
          </div>
          {{ $t("toolbar.show", { std: "STDERR" }) }}
        </a>
      </li>
      <li class="line"></li>
      <li>
        <a @click="showHostname = !showHostname">
          <mdi:check class="w-4" v-if="showHostname" />
          <div v-else class="w-4"></div>
          {{ $t("toolbar.show-hostname") }}
        </a>
      </li>
      <li>
        <a @click="showContainerName = !showContainerName">
          <mdi:check class="w-4" v-if="showContainerName" />
          <div v-else class="w-4"></div>
          {{ $t("toolbar.show-container-name") }}
        </a>
      </li>
    </ul>
  </div>
</template>

<script lang="ts" setup>
const { showSearch } = useSearchFilter();

const clear = defineEmit();

const { streamConfig, showHostname, showContainerName, containers } = useLoggingContext();

const downloadParams = computed(() =>
  Object.entries(toValue(streamConfig))
    .filter(([, value]) => value)
    .reduce((acc, [key]) => ({ ...acc, [key]: "1" }), {}),
);

const downloadUrl = computed(() =>
  withBase(
    `/api/containers/${containers.value.map((c) => c.host + "~" + c.id).join(",")}/download?${new URLSearchParams(downloadParams.value).toString()}`,
  ),
);

const hideMenu = (e: MouseEvent) => {
  if (e.target instanceof HTMLAnchorElement) {
    setTimeout(() => {
      if (document.activeElement instanceof HTMLElement) {
        document.activeElement.blur();
      }
    }, 50);
  }
};
</script>

<style scoped>
@reference "@/main.css";
li.line {
  @apply bg-base-content/20 h-px;
}

a {
  @apply whitespace-nowrap;
}
</style>
