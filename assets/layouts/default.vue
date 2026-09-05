<template>
  <div>
    <MobileMenu v-if="isMobile && !forceMenuHidden" @search="showFuzzySearch"></MobileMenu>
    <Splitpanes @resized="onResized($event)">
      <Pane min-size="10" :size="menuWidth" v-if="navVisible">
        <SidePanel />
      </Pane>
      <Pane min-size="10" :size="navVisible ? 100 - menuWidth : 100">
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
import { collapseNav } from "@/stores/settings";
import SideDrawer from "@/components/common/SideDrawer.vue";

const pinnedLogsStore = usePinnedLogsStore();
const { pinnedLogs } = storeToRefs(pinnedLogsStore);

const drawer = useTemplateRef<InstanceType<typeof SideDrawer>>("drawer") as Ref<InstanceType<typeof SideDrawer>>;
const { component: drawerComponent, properties: drawerProperties, width: drawerWidth } = createDrawer(drawer);

import { useFuzzySearch } from "@/composable/fuzzySearch";

// Pulls fuse.js (~48 KB) with it, and the palette only renders once the user opens it.
const FuzzySearchModal = defineAsyncComponent(() => import("@/components/FuzzySearchModal.vue"));

const modal = ref<HTMLDialogElement>();
const { open, openSearch: showFuzzySearch, closeSearch } = useFuzzySearch();
const searchParams = new URLSearchParams(window.location.search);
const forceMenuHidden = ref(searchParams.has("hideMenu"));

// splitpanes only reads a pane's `size` as its "given size" when the pane first
// registers, and its initial layout is only exact when every pane declares one.
// So both panes keep an explicit size, and this one tells the content pane to
// claim the whole width while the nav pane is unmounted — otherwise the two
// panes would together still claim 100% + menuWidth once the nav came back, and
// splitpanes would shrink the nav to its min-size to compensate.
const navVisible = computed(() => !isMobile.value && !collapseNav.value && !forceMenuHidden.value);

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

// splitpanes also emits `resized` for its own relayouts (the collapse toggle
// adding/removing the nav pane, a remount). Those carry neither `event` nor
// `index`, so only a width the user actually dragged to gets persisted.
function onResized({ panes, event, index }: { panes: { size: number }[]; event?: Event; index?: number }) {
  if (event === undefined && index === undefined) return;
  if (panes.length == 2) {
    menuWidth.value = Math.min(panes[0].size, 50);
  }
}
</script>

<style scoped>
@reference "@/main.css";

:deep(.splitpanes--vertical > .splitpanes__splitter) {
  @apply bg-base-100 relative min-w-[5px];
  transition: background-color 150ms ease-out;
}

/* Grab zone reaches a few pixels past the visible divider so the drag is
 * easier to start without making the divider itself any wider. */
:deep(.splitpanes--vertical > .splitpanes__splitter)::before {
  content: "";
  @apply absolute inset-y-0 -right-1 -left-1 z-20;
}

:deep(.splitpanes--vertical > .splitpanes__splitter):hover,
:deep(.splitpanes--dragging.splitpanes--vertical > .splitpanes__splitter) {
  @apply bg-secondary;
}

@media (prefers-reduced-motion: reduce) {
  :deep(.splitpanes--vertical > .splitpanes__splitter) {
    transition: none;
  }
}

@media screen and (max-width: 768px) {
  .router-view {
    padding-top: var(--mobile-nav-height);
  }
}
</style>
