<template>
  <div>
    <MobileMenu v-if="isMobile && !forceMenuHidden" @search="showFuzzySearch"></MobileMenu>
    <Splitpanes @resized="onResized($event)">
      <Pane min-size="10" :size="menuWidth" v-if="!isMobile && !collapseNav && !forceMenuHidden">
        <SidePanel @search="showFuzzySearch" />
      </Pane>
      <Pane min-size="10" :size="100 - menuWidth">
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
  <dialog ref="modal" class="modal bg-base-300/50! items-start backdrop-blur-md transition-none!" @close="open = false">
    <div class="modal-box max-w-2xl bg-transparent pt-20 shadow-none">
      <FuzzySearchModal @close="open = false" v-if="open" />
    </div>
    <form method="dialog" class="modal-backdrop">
      <button>close</button>
    </form>
  </dialog>
  <SideDrawer ref="drawer" :width="drawerWidth">
    <Suspense :timeout="0">
      <component :is="drawerComponent" v-bind="drawerProperties" />
      <template #fallback> Loading dependencies... </template>
    </Suspense>
  </SideDrawer>
  <ToastModal />
</template>

<script lang="ts" setup>
// @ts-ignore - splitpanes types are not available
import { Splitpanes, Pane } from "splitpanes";
import { collapseNav } from "@/stores/settings";
import SideDrawer from "@/components/common/SideDrawer.vue";

const pinnedLogsStore = usePinnedLogsStore();
const { pinnedLogs } = storeToRefs(pinnedLogsStore);

const drawer = useTemplateRef<InstanceType<typeof SideDrawer>>("drawer") as Ref<InstanceType<typeof SideDrawer>>;
const { component: drawerComponent, properties: drawerProperties, width: drawerWidth } = createDrawer(drawer);

const modal = ref<HTMLDialogElement>();
const open = ref(false);
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

function showFuzzySearch() {
  open.value = true;
}

function onResized({ panes }: { panes: { size: number }[] }) {
  if (panes.length == 2) {
    menuWidth.value = Math.min(panes[0].size, 50);
  }
}
</script>

<style scoped>
@reference "@/main.css";

:deep(.splitpanes--vertical > .splitpanes__splitter) {
  @apply bg-base-100 hover:bg-secondary min-w-[5px];
}

@media screen and (max-width: 768px) {
  .router-view {
    padding-top: 75px;
  }
}
</style>
