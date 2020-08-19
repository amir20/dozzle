<template>
  <log-container
    :id="id"
    :title="title"
    :show-title="activeContainers.length > 0"
    :scrollable="activeContainers.length > 0"
  >
  </log-container>
</template>

<script>
import { mapActions, mapGetters, mapState } from "vuex";

import LogContainer from "../components/LogContainer";
import store from "../store";

export default {
  props: ["id"],
  name: "Container",
  components: {
    LogContainer,
  },
  data() {
    return {
      title: "loading",
    };
  },
  metaInfo() {
    return {
      title: this.title,
    };
  },
  mounted() {
    if (this.allContainersById[this.id]) {
      this.title = this.allContainersById[this.id].name;
    }
  },
  computed: {
    ...mapState(["activeContainers"]),
    ...mapGetters(["allContainersById"]),
  },
  watch: {
    id() {
      this.title = this.allContainersById[this.id].name;
    },
    allContainersById() {
      this.title = this.allContainersById[this.id].name;
    },
  },
};
</script>
