<template>
  <dropdown-menu class="is-right">
    <a class="dropdown-item" @click="onClearClicked">
      <div class="level is-justify-content-start">
        <div class="level-left">
          <div class="level-item">
            <octicon-trash-24 class="mr-4" />
          </div>
        </div>
        <div class="level-right">
          <div class="level-item">Clear</div>
        </div>
      </div>
    </a>
    <a class="dropdown-item" :href="`${base}/api/logs/download?id=${container.id}`">
      <div class="level is-justify-content-start">
        <div class="level-left">
          <div class="level-item">
            <octicon-download-24 class="mr-4" />
          </div>
        </div>
        <div class="level-right">
          <div class="level-item">Download</div>
        </div>
      </div>
    </a>
    <hr class="dropdown-divider" />
    <a class="dropdown-item" @click="showSearch = true">
      <div class="level is-justify-content-start">
        <div class="level-left">
          <div class="level-item">
            <mdi-light-magnify class="mr-4" />
          </div>
        </div>
        <div class="level-right">
          <div class="level-item">Search</div>
        </div>
      </div>
    </a>
  </dropdown-menu>
</template>

<script lang="ts" setup>
import { onMounted, onUnmounted, PropType } from "vue";
import hotkeys from "hotkeys-js";
import config from "@/stores/config";
import { Container } from "@/types/Container";
import { useSearchFilter } from "@/composables/search";

const { showSearch } = useSearchFilter();

const { base } = config;

const props = defineProps({
  onClearClicked: {
    type: Function as PropType<(e: Event) => void>,
    default: (e: Event) => {},
  },
  container: {
    type: Object as () => Container,
    required: true,
  },
});

const onHotkey = (event: Event) => {
  props.onClearClicked(event);
  event.preventDefault();
};

onMounted(() => hotkeys("shift+command+l, shift+ctrl+l", onHotkey));
onUnmounted(() => hotkeys.unbind("shift+command+l, shift+ctrl+l", onHotkey));
</script>

<style lang="scss" scoped>
#download.button,
#clear.button {
  .icon {
    height: 80%;
  }

  &:hover {
    color: var(--primary-color);
    border-color: var(--primary-color);
  }
}
</style>
