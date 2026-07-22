<template>
  <div>
    <MobileMenu v-if="isMobile && !forceMenuHidden" @search="showFuzzySearch"></MobileMenu>
    <Splitpanes @resized="onResized($event)" :class="{ 'nav-collapsed': collapseNav }">
      <Pane
        :min-size="collapseNav ? 0 : MIN_MENU_WIDTH"
        :size="collapseNav ? 0 : menuWidth"
        v-if="!isMobile && !forceMenuHidden"
      >
        <SidePanel v-show="!collapseNav" />
      </Pane>
      <Pane :min-size="MIN_MENU_WIDTH" :size="collapseNav ? 100 : 100 - menuWidth">
        <Splitpanes>
          <Pane class="router-view min-h-screen">
            <router-view></router-view>
          </Pane>
          <template v-if="!isMobile">
            <Pane v-for="other in pinnedLogs" :key="other.id">
              <ContainerLog
                :id="other.id"
                show-title
                scrollable
                closable
                @close="pinnedLogsStore.unPinContainer(other)"
              />
            </Pane>
          </template>
        </Splitpanes>
      </Pane>
    </Splitpanes>
    <label
      class="btn btn-circle swap bg-base-content/10 swap-rotate border-base-content/20 hover:border-primary fixed bottom-4 -left-12 w-16 shadow-sm transition-all hover:-left-4"
      :class="{ '-left-6!': collapseNav }"
      v-if="!isMobile && !forceMenuHidden"
    >
      <input type="checkbox" v-model="collapseNav" />
      <mdi:chevron-right class="swap-on" />
      <mdi:chevron-left class="swap-off" />
    </label>
  </div>
  <dialog ref="modal" class="modal bg-base-300/50! items-start backdrop-blur-md transition-none!" @close="closeSearch">
    <div class="modal-box max-w-2xl overflow-visible! bg-transparent pt-20 shadow-none">
      <FuzzySearchModal @close="closeSearch" v-if="open" />
    </div>
    <form method="dialog" class="modal-backdrop">
      <button>close</button>
    </form>
  </dialog>
  <SideDrawer ref="drawer" :width="drawerWidth" v-slot="{ close }">
    <Suspense :timeout="0">
      <component :is="drawerComponent" v-bind="drawerProperties" :close="close" />
      <template #fallback> <span class="loading loading-spinner loading-sm"></span></template>
    </Suspense>
  </SideDrawer>
  <ToastModal />
</template>

<script lang="ts" setup>
import { Splitpanes, Pane } from "splitpanes";
import { collapseNav, MIN_MENU_WIDTH } from "@/stores/settings";
import SideDrawer from "@/components/common/SideDrawer.vue";

const pinnedLogsStore = usePinnedLogsStore();
const { pinnedLogs } = storeToRefs(pinnedLogsStore);

const drawer = useTemplateRef<InstanceType<typeof SideDrawer>>("drawer") as Ref<InstanceType<typeof SideDrawer>>;
const { component: drawerComponent, properties: drawerProperties, width: drawerWidth } = createDrawer(drawer);

import { useFuzzySearch } from "@/composable/fuzzySearch";

const modal = ref<HTMLDialogElement>();
const { open, openSearch: showFuzzySearch, closeSearch } = useFuzzySearch();
const searchParams = new URLSearchParams(window.location.search);
const forceMenuHidden = ref(searchParams.has("hideMenu"));

watch(open, () => {
  if (open.value) {
    modal.value?.showModal();
  } else {
    modal.value?.close();
  }
});

onKeyStroke("k", (e) => {
  if ((e.ctrlKey || e.metaKey) && !e.shiftKey) {
    showFuzzySearch();
    e.preventDefault();
  }
});

function onResized({ panes }: { panes: { size: number }[] }) {
  // Ignore the resize that collapsing/expanding triggers; only persist drags.
  if (collapseNav.value) return;
  if (panes.length == 2) {
    menuWidth.value = Math.min(panes[0].size, 50);
  }
}
</script>

<style scoped>
@reference "@/main.css";

:deep(.splitpanes--vertical > .splitpanes__splitter) {
  @apply bg-base-100 hover:bg-secondary min-w-[5px];
  transition: opacity 0.3s cubic-bezier(0.2, 0, 0, 1);
}

/* Hide (and disable) the resize handle while the sidebar is collapsed. */
:deep(.splitpanes.nav-collapsed > .splitpanes__splitter) {
  opacity: 0;
  pointer-events: none;
}

@media screen and (max-width: 768px) {
  .router-view {
    padding-top: var(--mobile-nav-height);
  }
}
</style>
