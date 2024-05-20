<template>
  <div>
    <mobile-menu v-if="isMobile && !forceMenuHidden" @search="showFuzzySearch"></mobile-menu>
    <splitpanes @resized="onResized($event)">
      <pane min-size="10" :size="menuWidth" v-if="!isMobile && !collapseNav && !forceMenuHidden">
        <side-panel @search="showFuzzySearch"></side-panel>
      </pane>
      <pane min-size="10">
        <splitpanes>
          <pane class="router-view min-h-screen">
            <router-view></router-view>
          </pane>
          <template v-if="!isMobile">
            <pane v-for="other in pinnedLogs" :key="other.id">
              <ContainerLog
                :id="other.id"
                show-title
                scrollable
                closable
                @close="pinnedLogsStore.unPinContainer(other)"
              />
            </pane>
          </template>
        </splitpanes>
      </pane>
    </splitpanes>
    <label
      class="btn btn-circle swap btn-neutral swap-rotate fixed -left-12 bottom-4 w-16 transition-all hover:-left-4"
      :class="{ '!-left-6': collapseNav }"
      v-if="!isMobile && !forceMenuHidden"
    >
      <input type="checkbox" v-model="collapseNav" />
      <mdi:chevron-right class="swap-on" />
      <mdi:chevron-left class="swap-off" />
    </label>
  </div>
  <dialog ref="modal" class="modal items-start bg-white/20 backdrop:backdrop-blur-sm" @close="open = false">
    <div class="modal-box max-w-2xl bg-transparent pt-20 shadow-none">
      <FuzzySearchModal @close="open = false" v-if="open" />
    </div>
    <form method="dialog" class="modal-backdrop">
      <button>close</button>
    </form>
  </dialog>
  <div class="toast toast-end whitespace-normal">
    <div
      class="alert max-w-xl"
      v-for="toast in toasts"
      :key="toast.id"
      :class="{
        'alert-error': toast.type === 'error',
        'alert-info': toast.type === 'info',
        'alert-warning': toast.type === 'warning',
      }"
    >
      <carbon:information class="size-6 shrink-0 stroke-current" v-if="toast.type === 'info'" />
      <carbon:warning class="size-6 shrink-0 stroke-current" v-else-if="toast.type === 'error'" />
      <carbon:warning class="size-6 shrink-0 stroke-current" v-else-if="toast.type === 'warning'" />
      <div>
        <h3 class="text-lg font-bold" v-if="toast.title">{{ toast.title }}</h3>
        <div v-html="toast.message" class="[&>a]:underline"></div>
      </div>
      <div>
        <button class="btn btn-circle btn-xs" @click="removeToast(toast.id)"><mdi:close /></button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
// @ts-ignore - splitpanes types are not available
import { Splitpanes, Pane } from "splitpanes";
import { collapseNav } from "@/stores/settings";

const pinnedLogsStore = usePinnedLogsStore();
const { pinnedLogs } = storeToRefs(pinnedLogsStore);

const { toasts, removeToast } = useToast();

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

<style scoped lang="postcss">
:deep(.splitpanes--vertical > .splitpanes__splitter) {
  @apply min-w-[3px] bg-base-lighter hover:bg-secondary;
}

@media screen and (max-width: 768px) {
  .router-view {
    padding-top: 75px;
  }
}
</style>
