<template>
  <div>
    <search></search>
    <log-container :id="id" show-title :scrollable="activeContainers.length > 0"> </log-container>
  </div>
</template>

<script>
import { mapGetters } from "vuex";
import Search from "../components/Search.vue";
import LogContainer from "../components/LogContainer.vue";

export default {
  props: ["id"],
  name: "Container",
  components: {
    LogContainer,
    Search,
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
    ...mapGetters(["allContainersById", "activeContainers"]),
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
