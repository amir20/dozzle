<template>
  <scrollable-view :scrollable="scrollable" v-if="container">
    <template v-slot:header v-if="showTitle">
      <div class="mr-0 columns is-vcentered is-marginless is-hidden-mobile">
        <div class="column is-clipped is-paddingless">
          <container-title :value="container.name" @close="$emit('close')"></container-title>
        </div>
        <div class="column is-clipped is-paddingless">
          <container-stat :stat="container.stat" :state="container.state"></container-stat>
        </div>
        <div class="column is-clipped is-paddingless"></div>
        <div class="column is-narrow is-paddingless" v-if="closable">
          <button class="delete is-medium" @click="$emit('close')"></button>
        </div>
      </div>
      <div class="mr-0 is-vcentered is-hidden-mobile">
        <div class="is-flex">
          <a @click="onClearClicked" class="button is-small is-outlined" id="clear">
            <span class="icon">
              <icon name="bin"></icon>
            </span>
          </a>
          <div class="is-flex-grow-1"></div>
          <a
            class="button is-small is-outlined"
            id="download"
            :href="`${base}/api/logs/download?id=${container.id}`"
            download
          >
            <span class="icon">
              <icon name="save"></icon>
            </span>
          </a>
        </div>
      </div>
    </template>
    <template v-slot="{ setLoading }">
      <log-viewer-with-source ref="logViewer" :id="id" @loading-more="setLoading($event)"></log-viewer-with-source>
    </template>
  </scrollable-view>
</template>

<script>
import LogViewerWithSource from "./LogViewerWithSource";
import ScrollableView from "./ScrollableView";
import ContainerTitle from "./ContainerTitle";
import ContainerStat from "./ContainerStat";
import Icon from "./Icon";
import config from "../store/config";
import containerMixin from "./mixins/container";

export default {
  mixins: [containerMixin],
  props: {
    id: {
      type: String,
    },
    showTitle: {
      type: Boolean,
      default: false,
    },
    scrollable: {
      type: Boolean,
      default: false,
    },
    closable: {
      type: Boolean,
      default: false,
    },
  },
  name: "LogContainer",
  components: {
    LogViewerWithSource,
    ScrollableView,
    ContainerTitle,
    ContainerStat,
    Icon,
  },
  computed: {
    base() {
      return config.base;
    },
  },
  methods: {
    onClearClicked() {
      this.$refs.logViewer.clear();
    },
  },
};
</script>
<style lang="scss" scoped>
button.delete {
  background-color: var(--scheme-main-ter);
  opacity: 0.6;
  &:after,
  &:before {
    background-color: var(--text-color);
  }

  &:hover {
    opacity: 1;
  }
}

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
