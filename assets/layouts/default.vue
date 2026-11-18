<template>
  <!-- Behind an open cloud panel the page is inert, so Tab stays in the panel and
       the rail that is its tab bar. Toasts and the rail live outside this div. -->
  <div :inert="(railAvailable && !!cloudPanel) || undefined">
    <MobileTabBar v-if="showTabBar" @search="showFuzzySearch" />
    <Splitpanes @resized="onResized($event)">
      <Pane :min-size="navMinSize" :size="navSize" v-if="navVisible">
        <SidePanel />
      </Pane>
      <Pane min-size="10" :size="navVisible ? 100 - navSize : 100">
        <!-- The strip is fixed to the right edge, so its width is reserved here
             rather than over the logs. Reserved inside the content column and
             not on the page, because the nav is a different column. -->
        <div :style="{ paddingRight: `${railOffset}px` }">
          <Splitpanes>
            <Pane class="router-view min-h-screen" :class="{ 'tabbar-clearance': showTabBar }">
              <router-view></router-view>
            </Pane>
            <template v-if="!isMobile">
              <Pane v-for="other in visiblePinnedLogs" :key="other.id">
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
        </div>
      </Pane>
    </Splitpanes>
    <!-- Hiding lives at the bottom of the sidebar itself; this is what brings it back,
         the same edge tab the cloud strip leaves on the other side. -->
    <EdgeTab
      v-if="!isMobile && !forceMenuHidden && collapseNav"
      side="left"
      :label="$t('label.show-sidebar')"
      @click="collapseNav = false"
    />
  </div>
  <!-- Before the rail, so at the same layer the rail stays on top: the drawer
       slides out from behind it. -->
  <CloudDrawer v-if="railAvailable" />
  <CloudRail v-if="railMounted" />
  <CloudRailHandle v-else-if="railCollapsed" />
  <WelcomeModal v-if="cloudSurfaceMounted" />
  <!-- A dialog sits in the top layer, so the body's safe-area padding never reaches it. On a phone
       with a side status column (iPhone Duo) the palette's close button ended up underneath it. -->
  <dialog
    ref="modal"
    class="modal bg-base-300/50! items-start pr-[env(safe-area-inset-right)] pl-[env(safe-area-inset-left)] backdrop-blur-md transition-none!"
    @close="closeSearch"
  >
    <div class="modal-box max-w-3xl overflow-visible! bg-transparent pt-20 shadow-none">
      <FuzzySearchModal @close="closeSearch" v-if="open" />
    </div>
    <form method="dialog" class="modal-backdrop">
      <button>close</button>
    </form>
  </dialog>
  <SideDrawer ref="drawer" :width="drawerWidth" :close-guard="drawerCloseGuard" v-slot="{ close }">
    <Suspense :timeout="0">
      <component :is="drawerComponent" v-bind="drawerProperties" :close="close" />
      <template #fallback> <span class="loading loading-spinner loading-sm"></span></template>
    </Suspense>
  </SideDrawer>
  <!-- The phone's cloud sheet floats its own tab bar at the bottom edge. -->
  <ToastModal :above-tab-bar="showTabBar || (isMobile && !!cloudPanel)" />
  <SetupWizard />
</template>

<script lang="ts" setup>
import { Splitpanes, Pane } from "splitpanes";
import { collapseNav } from "@/stores/settings";
import SideDrawer from "@/components/shell/SideDrawer.vue";

usePinnedColumnsInUrl();
const pinnedLogsStore = usePinnedLogsStore();
const visiblePinnedLogs = useVisiblePinnedLogs();

const drawer = useTemplateRef<InstanceType<typeof SideDrawer>>("drawer") as Ref<InstanceType<typeof SideDrawer>>;
const {
  component: drawerComponent,
  properties: drawerProperties,
  width: drawerWidth,
  closeGuard: drawerCloseGuard,
} = createDrawer(drawer);

import { useFuzzySearch } from "@/composable/app/fuzzySearch";

// Pulls fuse.js (~48 KB) with it, and the palette only renders once the user opens it.
const FuzzySearchModal = defineAsyncComponent(() => import("@/components/search/FuzzySearchModal.vue"));
// Only rendered for pinned columns; the route's own view imports it when it needs it.
const ContainerLog = defineAsyncComponent(() => import("@/components/views/ContainerLog.vue"));

const {
  panel: cloudPanel,
  available: railAvailable,
  mounted: railMounted,
  collapsed: railCollapsed,
  railOffset,
  toggleRail,
  openRail,
  closeRail,
} = useCloudRail();

// The cloud panels have a drawer of their own beside the rail. Every way in
// (the rail, the chord, a toolbar, a log row's menu) comes through here, so this
// is the one gate on where the panels can show.
watch(cloudPanel, (next, prev) => {
  if (!next || prev) return;
  if (!railAvailable.value) return closeRail();
  // The app's drawer is modal and would sit over the panel, so opening one
  // closes it, and that close goes through its guard: a half-built alert form
  // asks before it is thrown away.
  const guard = drawerCloseGuard.value;
  if (guard && !guard()) return closeRail();
  drawer.value?.close();
});
// Unlinking cloud leaves the panels with nothing to show.
watch(railAvailable, (ok) => ok || closeRail());

const modal = ref<HTMLDialogElement>();
const { open, openSearch: showFuzzySearch, closeSearch } = useFuzzySearch();
const { mounted: cloudSurfaceMounted } = useCloudSurface();
const searchParams = new URLSearchParams(window.location.search);

onMounted(startUsageReporting);
const forceMenuHidden = ref(searchParams.has("hideMenu"));

// splitpanes only reads a pane's `size` as its "given size" when the pane first
// registers, and its initial layout is only exact when every pane declares one.
// So both panes keep an explicit size, and this one tells the content pane to
// claim the whole width while the nav pane is unmounted — otherwise the two
// panes would together still claim 100% + menuWidth once the nav came back, and
// splitpanes would shrink the nav to its min-size to compensate.
const navVisible = computed(() => !isMobile.value && !collapseNav.value && !forceMenuHidden.value);

// menuWidth is a percentage, so on a narrow desktop (an unfolded phone, an iPad) the
// default 15% left about 130px and every container name read "doli…". Floor it in pixels.
const NAV_MIN_PX = 200;
const { width: windowWidth } = useWindowSize();
const navMinSize = computed(() => Math.min(40, Math.max(10, (NAV_MIN_PX / windowWidth.value) * 100)));
const navSize = computed(() => Math.max(menuWidth.value, navMinSize.value));

// On a phone the shell is a tab bar for the top-level pages, and a log view is pushed on
// top of it with a back button instead: the bottom of the stream is where people read.
const route = useRoute();
const onLogRoute = computed(() => isLogRoute(route));
const showTabBar = computed(() => isMobile.value && !forceMenuHidden.value && !onLogRoute.value);
useEdgeSwipeBack(computed(() => isMobile.value && onLogRoute.value));

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

// Shift is the assistant; the palette keeps the bare chord. Registered on the
// layout rather than the toolbar so it works on every view it can talk about.
onKeyStroke(["k", "K"], (e) => {
  if ((e.ctrlKey || e.metaKey) && e.shiftKey) {
    if (railAvailable.value) toggleRail("chat");
    e.preventDefault();
  }
});
// The bare letter the log toolbars hint at, here for the same reason. Only where
// the rail can show, so a page without one never has it un-hidden behind its back.
useLetterShortcut(".", () => openRail("chat"), railAvailable);

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

.tabbar-clearance {
  padding-bottom: var(--mobile-tabbar-height);
}
</style>
