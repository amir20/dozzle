<template>
  <div class="toolbar mr-0 is-vcentered is-hidden-mobile">
    <div class="is-flex">
      <b-tooltip type="is-dark" label="Clear">
        <a @click="onClearClicked" class="button is-small is-light is-inverted pl-1 pr-1" id="clear">
          <clear-icon />
        </a>
      </b-tooltip>
      <div class="is-flex-grow-1"></div>
      <b-tooltip type="is-dark" label="Download">
        <a
          class="button is-small is-light is-inverted pl-1 pr-1"
          id="download"
          :href="`${base}/api/logs/download?id=${container.id}`"
          download
        >
          <download-icon />
        </a>
      </b-tooltip>
    </div>
  </div>
</template>

<script>
import config from "../store/config";
import hotkeys from "hotkeys-js";
import DownloadIcon from "~icons/octicon/download-24";
import ClearIcon from "~icons/octicon/trash-24";

export default {
  props: {
    onClearClicked: {
      type: Function,
      default: () => {},
    },
    container: {
      type: Object,
    },
  },
  name: "LogActionsToolbar",
  components: {
    DownloadIcon,
    ClearIcon,
  },
  computed: {
    base() {
      return config.base;
    },
  },
  mounted() {
    hotkeys("shift+command+l, shift+ctrl+l", (event, handler) => {
      this.onClearClicked();
      event.preventDefault();
    });
  },
};
</script>

<style>
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
