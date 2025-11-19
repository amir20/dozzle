<template>
  <div class="dropdown dropdown-end dropdown-hover z-20">
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
        <a @click="clear()">
          <octicon:trash-24 /> {{ $t("toolbar.clear") }}
          <KeyShortcut char="k" :modifiers="['shift', 'meta']" />
        </a>
      </li>
      <li v-if="enableDownload">
        <a :href="downloadUrl" download>
          <octicon:download-24 />
          {{ isFiltered ? $t("toolbar.download-filtered") : $t("toolbar.download") }}
        </a>
      </li>
      <li>
        <a @click="showSearch = true">
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
import { allLevels } from "@/composable/logContext";

const { showSearch, debouncedSearchFilter } = useSearchFilter();
const { enableDownload } = config;
const clear = defineEmit();

const { streamConfig, showHostname, showContainerName, containers, levels } = useLoggingContext();

const downloadParams = computed(() => {
  const params = Object.entries(toValue(streamConfig))
    .filter(([, value]) => value)
    .reduce((acc, [key]) => ({ ...acc, [key]: "1" }), {} as Record<string, string>);

  // Add filter if search is active
  if (debouncedSearchFilter.value) {
    params.filter = debouncedSearchFilter.value;
  }

  // Add selected levels
  const selectedLevels = Array.from(levels.value);
  if (selectedLevels.length > 0 && selectedLevels.length < allLevels.length) {
    selectedLevels.forEach((level) => {
      params[`levels`] = level;
    });
  }

  return params;
});

const downloadUrl = computed(() => {
  const params = new URLSearchParams();
  const downloadParamsValue = downloadParams.value;

  // Add stdout/stderr
  if (downloadParamsValue.stdout) params.append("stdout", "1");
  if (downloadParamsValue.stderr) params.append("stderr", "1");

  // Add filter
  if (downloadParamsValue.filter) params.append("filter", downloadParamsValue.filter);

  // Add levels (multiple values)
  const selectedLevels = Array.from(levels.value);
  if (selectedLevels.length > 0 && selectedLevels.length < allLevels.length) {
    selectedLevels.forEach((level) => params.append("levels", level));
  }

  return withBase(
    `/api/containers/${containers.value.map((c) => c.host + "~" + c.id).join(",")}/download?${params.toString()}`,
  );
});

const isFiltered = computed(
  () => debouncedSearchFilter.value || (levels.value.size > 0 && levels.value.size < allLevels.length),
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
