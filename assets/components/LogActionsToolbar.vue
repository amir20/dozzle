<template>
  <div class="mr-2 column is-narrow is-paddingless is-clickable">
    <o-dropdown aria-role="list" position="bottom-left">
      <template v-slot:trigger>
        <span class="btn">
          <span class="icon">
            <mdi-dots-vertical />
          </span>
        </span>
      </template>

      <o-dropdown-item aria-role="listitem" @click="onClearClicked">
        <octicon-trash-24 style="margin-right: 1em" />
        <span> Clear </span>
      </o-dropdown-item>
      <a id="download" :href="`${base}/api/logs/download?id=${container.id}`">
        <o-dropdown-item aria-role="listitem">
          <octicon-download-24 style="margin-right: 1em" />
          <span> Download </span>
        </o-dropdown-item>
      </a>
    </o-dropdown>
  </div>
</template>

<script lang="ts" setup>
import config from "@/stores/config";
import { Container } from "@/types/Container";
import hotkeys from "hotkeys-js";
import { onMounted, onUnmounted } from "vue";

const props = defineProps({
  onClearClicked: {
    type: Function,
    default: () => {},
  },
  container: {
    type: Object as () => Container,
    required: true,
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
