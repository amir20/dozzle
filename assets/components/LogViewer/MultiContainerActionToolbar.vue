<template>
  <div
    class="dropdown dropdown-end z-20"
    :class="{ 'dropdown-hover': canHover }"
    @mouseenter="actionsMenuOpen = true"
    @mouseleave="onLeave"
  >
    <div
      tabindex="0"
      role="button"
      class="btn btn-ghost btn-sm w-8 px-0"
      :aria-label="$t('toolbar.more-actions')"
      @click="focusOnTap"
    >
      <mdi:dots-horizontal class="size-6" />
    </div>
    <ul
      tabindex="0"
      class="menu dropdown-content rounded-box bg-base-200 border-base-content/20 z-50 w-52 border p-1 shadow-sm"
      @click="hideMenu"
    >
      <li>
        <a @click="clear()">
          <octicon:trash-24 /> {{ $t("toolbar.clear") }}
          <KeyShortcut char="l" :modifiers="['shift', 'meta']" />
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
      <li class="line"></li>
      <li v-if="!isMobile">
        <a @click="resetMenuWidth" :class="{ 'pointer-events-none opacity-40': !canResetMenuWidth }">
          <mdi:arrow-collapse-horizontal />
          {{ $t("toolbar.reset-sidebar-width") }}
        </a>
      </li>
      <li>
        <router-link v-if="!settingsAsPopup" :to="{ name: '/settings' }">
          <mdi:cog />
          {{ $t("title.settings") }}
        </router-link>
        <a v-else @click="openSettings">
          <mdi:cog />
          {{ $t("title.settings") }}
        </a>
      </li>
    </ul>
  </div>
</template>

<script lang="ts" setup>
const { showSearch } = useSearchFilter();
const { actionsMenuOpen, canHover, focusOnTap, hideMenu, onLeave } = useActionsMenu();
const { openSettings } = useSettingsModal();
const { enableDownload } = config;
const clear = defineEmit();

const { name } = defineProps<{ name?: string }>();

const { streamConfig, showHostname, showContainerName, containers, levels } = useLoggingContext();

const { downloadUrl, isFiltered } = useDownloadUrl(containers, streamConfig, levels, name);
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
