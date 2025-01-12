<template>
  <div>
    <mobile-menu v-if="isMobile && !forceMenuHidden" @search="showFuzzySearch"></mobile-menu>
    <Splitpanes @resized="onResized($event)">
      <Pane min-size="10" :size="menuWidth" v-if="!isMobile && !collapseNav && !forceMenuHidden">
        <SidePanel @search="showFuzzySearch" />
      </Pane>
      <Pane min-size="10">
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
      class="btn btn-circle swap bg-base-100 swap-rotate fixed bottom-4 -left-12 w-16 transition-all hover:-left-4"
      :class="{ '-left-6!': collapseNav }"
      v-if="!isMobile && !forceMenuHidden"
    >
      <input type="checkbox" v-model="collapseNav" />
      <mdi:chevron-right class="swap-on" />
      <mdi:chevron-left class="swap-off" />
    </label>
  </div>
  <dialog
    ref="modal"
    class="modal items-start bg-white/20 transition-none backdrop:backdrop-blur-xs"
    @close="open = false"
  >
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

function onResized(e: any) {
  if (e.length == 2) {
    menuWidth.value = e[0].size;
  }
}
</script>

<style scoped>
@import "@/main.css" reference;

:deep(.splitpanes--vertical > .splitpanes__splitter) {
  @apply bg-base-100 hover:bg-secondary min-w-[3px];
}

@media screen and (max-width: 768px) {
  .router-view {
    padding-top: 75px;
  }
}
</style>
