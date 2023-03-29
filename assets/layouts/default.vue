<template>
  <main v-if="!authorizationNeeded">
    <mobile-menu v-if="isMobile" @search="showFuzzySearch"></mobile-menu>
    <splitpanes @resized="onResized($event)">
      <pane min-size="10" :size="menuWidth" v-if="!isMobile && !collapseNav">
        <side-menu @search="showFuzzySearch"></side-menu>
      </pane>
      <pane min-size="10">
        <splitpanes>
          <pane class="has-min-height router-view">
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
import { useProgrammatic } from "@oruga-ui/oruga-next";
import FuzzySearchModal from "@/components/FuzzySearchModal.vue";

const { oruga } = useProgrammatic();
const { authorizationNeeded } = config;

const containerStore = useContainerStore();
const { activeContainers, visibleContainers } = storeToRefs(containerStore);

watchEffect(() => {
  setTitle(`${visibleContainers.value.length} containers`);
});

onKeyStroke("k", (e) => {
  if (e.ctrlKey || e.metaKey) {
    showFuzzySearch();
    e.preventDefault();
  }
});

function showFuzzySearch() {
  oruga.modal.open({
    // parent: this,
    component: FuzzySearchModal,
    animation: "false",
    width: 600,
    active: true,
  });
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

<style scoped lang="scss">
:deep(.splitpanes--vertical > .splitpanes__splitter) {
  min-width: 3px;
  background: var(--border-color);

  &:hover {
    background: var(--border-hover-color);
  }
}

@media screen and (max-width: 768px) {
  .router-view {
    padding-top: 75px;
  }
}

.button.has-no-border {
  border-color: transparent !important;
}

.has-min-height {
  min-height: 100vh;
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
