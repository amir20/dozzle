<template>
  <div class="dropdown is-right is-hoverable">
    <div class="dropdown-trigger">
      <button class="button" aria-haspopup="true" aria-controls="dropdown-menu">
        <span class="icon">
          <mdi-dots-vertical />
        </span>
      </button>
    </div>
    <div class="dropdown-menu" id="dropdown-menu" role="menu">
      <div class="dropdown-content">
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
      </div>
    </div>
  </div>
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

.toolbar {
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  width: 200px;
  background-color: var(--action-toolbar-background-color);
  border-radius: 8em;
  margin-top: 0.5em;

  & > div {
    margin: 0 2em;
    padding: 0.5em 0;
  }

  .button {
    background-color: rgba(0, 0, 0, 0) !important;

    &.is-small {
      font-size: 0.65rem;
    }
  }
}
</style>
