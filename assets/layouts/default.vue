<template>
  <main v-if="!authorizationNeeded">
    <dialog ref="modal" class="modal items-start bg-white/20 backdrop:backdrop-blur-sm">
      <div class="modal-box bg-transparent pt-20 shadow-none">
        <FuzzySearchModal @close="open = false" v-if="open" />
      </div>
      <form method="dialog" class="modal-backdrop">
        <button>close</button>
      </form>
    </dialog>
    <mobile-menu v-if="isMobile" @search="showFuzzySearch"></mobile-menu>
    <splitpanes @resized="onResized($event)">
      <pane min-size="10" :size="menuWidth" v-if="!isMobile && !collapseNav">
        <side-panel @search="showFuzzySearch"></side-panel>
      </pane>
      <pane min-size="10">
        <splitpanes>
          <pane class="router-view min-h-screen">
            <router-view></router-view>
          </pane>
          <template v-if="!isMobile">
            <pane v-for="other in activeContainers" :key="other.id">
              <log-container
                :id="other.id"
                show-title
                scrollable
                closable
                @close="containerStore.removeActiveContainer(other)"
              ></log-container>
            </pane>
          </template>
        </splitpanes>
      </pane>
    </splitpanes>
    <button
      @click="collapse"
      class="button is-small is-rounded"
      :class="{ collapsed: collapseNav }"
      id="hide-nav"
      v-if="!isMobile"
    >
      <span class="icon ml-2" v-if="collapseNav">
        <mdi:light-chevron-right />
      </span>
      <span class="icon" v-else>
        <mdi:light-chevron-left />
      </span>
    </button>
  </main>
</template>

<script lang="ts" setup>
// @ts-ignore - splitpanes types are not available
import { Splitpanes, Pane } from "splitpanes";
const { authorizationNeeded } = config;

const containerStore = useContainerStore();
const { activeContainers } = storeToRefs(containerStore);

const modal = ref<HTMLDialogElement>();
const open = ref(false);

useEventListener(modal, "close", () => (open.value = false));
whenever(open, () => modal.value?.showModal());
whenever(logicNot(open), () => modal.value?.close());

onKeyStroke("k", (e) => {
  if ((e.ctrlKey || e.metaKey) && !e.shiftKey) {
    showFuzzySearch();
    e.preventDefault();
  }
});

function showFuzzySearch() {
  open.value = true;
}

function collapse() {
  collapseNav.value = !collapseNav.value;
}
function onResized(e: any) {
  if (e.length == 2) {
    menuWidth.value = e[0].size;
  }
}
</script>

<style scoped lang="postcss">
:deep(.splitpanes--vertical > .splitpanes__splitter) {
  @apply min-w-[3px] bg-base-lighter hover:bg-secondary;
}

@media screen and (max-width: 768px) {
  .router-view {
    padding-top: 75px;
  }
}

#hide-nav {
  position: fixed;
  left: 10px;
  bottom: 10px;

  &.collapsed {
    left: -40px;
    width: 60px;
    padding-left: 40px;
    color: var(--text-strong-color);
    background: var(--scheme-main);

    &:hover {
      left: -25px;
    }
  }
}
</style>
