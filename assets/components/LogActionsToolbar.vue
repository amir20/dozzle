<template>
  <div class="mr-0 toolbar is-vcentered is-hidden-mobile">
    <div class="is-flex">
      <o-tooltip type="is-dark" label="Clear">
        <a @click="onClearClicked" class="pl-1 pr-1 button is-small is-light is-inverted" id="clear">
          <octicon-trash-24 />
        </a>
      </o-tooltip>
      <div class="is-flex-grow-1"></div>
      <o-tooltip type="is-dark" label="Download">
        <a
          class="pl-1 pr-1 button is-small is-light is-inverted"
          id="download"
          :href="`${base}/api/logs/download?id=${container.id}`"
          download
        >
          <octicon-download-24 />
        </a>
      </o-tooltip>
    </div>
  </div>
</template>

<script lang="ts" setup>
import config from "../store/config";
import hotkeys from "hotkeys-js";
import { onMounted, onUnmounted } from "vue";

const props = defineProps({
  onClearClicked: {
    type: Function,
    default: () => {},
  },
  container: {
    type: Object,
  },
});

const { base } = config;

const onHotkey = (event: Event) => {
  props.onClearClicked();
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
